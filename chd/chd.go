// A Go implementation of minimal perfect hashing (MPH).
//
// This package implements the compress, hash and displace (CHD) algorithm
// described here: http://cmph.sourceforge.net/papers/esa09.pdf
//
// See https://github.com/alecthomas/mph for source
package chd

import (
	"bytes"
	"encoding/binary"
	"github.com/alecthomas/mph"
	"github.com/alecthomas/unsafeslice"
	"io"
	"io/ioutil"
)

// CHD hash table lookup.
type CHD struct {
	// Random hash function table.
	r []uint64
	// Array of indices into hash function table r
	indices []uint16
	// Final table of values.
	keys   [][]byte
	values [][]byte
}

// Read a serialized CHD.
func Read(r io.Reader) (*CHD, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return Mmap(b)
}

type byteSliceIterator struct {
	b          []byte
	start, end uint64
}

func (b *byteSliceIterator) Read(size uint64) []byte {
	b.start, b.end = b.end, b.end+size
	return b.b[b.start:b.end]
}

func (b *byteSliceIterator) ReadInt() uint64 {
	return uint64(binary.LittleEndian.Uint32(b.Read(4)))
}

// Alias the CHD structure over an existing byte region (typically mmapped).
func Mmap(b []byte) (*CHD, error) {
	c := &CHD{}

	bi := &byteSliceIterator{b: b}

	// Read vector of hash functions.
	rl := bi.ReadInt()
	c.r = unsafeslice.Uint64SliceFromByteSlice(bi.Read(rl * 8))

	il := bi.ReadInt()
	c.indices = unsafeslice.Uint16SliceFromByteSlice(bi.Read(il * 2))

	el := bi.ReadInt()

	c.keys = make([][]byte, el)
	c.values = make([][]byte, el)

	for i := uint64(0); i < el; i++ {
		kl := bi.ReadInt()
		vl := bi.ReadInt()
		c.keys[i] = bi.Read(kl)
		c.values[i] = bi.Read(vl)
	}

	return c, nil
}

// Get an entry from the hash table.
func (c *CHD) Get(key []byte) []byte {
	r0 := c.r[0]
	h := chdHash(key) ^ r0
	i := h % uint64(len(c.indices))
	ri := c.indices[i]
	// This can happen if there were unassigned slots in the hash table.
	if ri >= uint16(len(c.r)) {
		return nil
	}
	r := c.r[ri]
	ti := (h ^ r) % uint64(len(c.keys))
	// fmt.Printf("r[0]=%d, h=%d, i=%d, ri=%d, r=%d, ti=%d\n", c.r[0], h, i, ri, r, ti)
	k := c.keys[ti]
	if bytes.Compare(k, key) != 0 {
		return nil
	}
	v := c.values[ti]
	return v
}

func (c *CHD) Len() int {
	return len(c.keys)
}

// Iterate over entries in the hash table.
func (c *CHD) Iterate() mph.Iterator {
	if len(c.keys) == 0 {
		return nil
	}
	return &chdIterator{c: c}
}

// Serialize the CHD. The serialized form is conducive to mmapped access. See
// the Mmap function for details.
func (c *CHD) Write(w io.Writer) error {
	write := func(nd ...interface{}) error {
		for _, d := range nd {
			if err := binary.Write(w, binary.LittleEndian, d); err != nil {
				return err
			}
		}
		return nil
	}

	data := []interface{}{
		uint32(len(c.r)), c.r,
		uint32(len(c.indices)), c.indices,
		uint32(len(c.keys)),
	}

	if err := write(data...); err != nil {
		return err
	}

	for i := range c.keys {
		k, v := c.keys[i], c.values[i]
		if err := write(uint32(len(k)), uint32(len(v))); err != nil {
			return err
		}
		if _, err := w.Write(k); err != nil {
			return err
		}
		if _, err := w.Write(v); err != nil {
			return err
		}
	}
	return nil
}

type chdIterator struct {
	i int
	c *CHD
}

func (c *chdIterator) Get() mph.Entry {
	return &chdEntry{key: c.c.keys[c.i], value: c.c.values[c.i]}
}

func (c *chdIterator) Next() mph.Iterator {
	c.i++
	if c.i >= len(c.c.keys) {
		return nil
	}
	return c
}
