# Minimal Perfect Hashing for Go

This library provides [Minimal Perfect Hashing](http://en.wikipedia.org/wiki/Perfect_hash_function) (MPH) using the [Compress, Hash and Displace](http://cmph.sourceforge.net/papers/esa09.pdf) (CHD) algorithm.

## What is this useful for?

Primarily, extremely efficient access to potentially very large static datasets, such as geographical data, NLP data sets, etc.

On my 2012 vintage MacBook Air, a benchmark against a wikipedia index with 300K keys against a 2GB TSV dump takes about ~200ns per lookup.

## How would it be used?

Typically, the table would be used as a fast index into a (much) larger data set, with values in the table being file offsets or similar.

The tables can be serialized. Numeric values are written in little endian form.

## Example code

Building and serializing an MPH hash table (error checking omitted for clarity):

```go
b := mph.Builder()
for k, v := range data {
    b.Add(k, v)
}
h, _ := b.Build()
w, _ := os.Create("data.idx")
b, _ := h.Write(w)
```

Deserializing the hash table and performing lookups:

```go
r, _ := os.Open("data.idx")
h, _ := h.Read(r)

v := h.Get([]byte("some key"))
if v == nil {
    // Key not found
}
```

MMAP is also indirectly supported, by deserializing from a byte slice and slicing the keys and values.

The [API documentation](http://godoc.org/github.com/alecthomas/mph) has more details.
