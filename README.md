## What is it?
The Black-White Array (aka **BWArr**) is a fast data structure based on arrays with *O(log N)* memory allocations.
This repository contains Go implementation.

### Key features:
- Fast insert, delete, and search operations *O(log N)* time amortized complexity;
- *O(log N)* memory allocations for Inserts - no pressure on GC;
- Array-based and pointerless makes it CPU-friendly: cache Locality / sequential iteration / etc;
- Drop-in replacement for `github.com/google/btree` and `github.com/petar/GoLLRB`;
- Low memory overhead - no pointers per element, compact memory representation;
- Batch-friendly: arrays under the hood allow efficient bulk operations (work in progress);
- Easily serializable;

### Tradeoffs
- One per *N* insert operations complexity falls down to *O(N)*, amortized remains *O(log N)*. For real-time systems, it may introduce latency spikes for collections with millions of elements.
- For some rare cases with deleting special series of elements, `Search()` operations may degrade to *O(N)/4*. Can be prevented by calling Compact();

## The Author 
The idea of Black-White Array was invented and published by [Z. George Mou](https://www.cs.brandeis.edu/~mou/) in [Black-White Array: A New Data Structure for
Dynamic Data Sets](https://arxiv.org/abs/2004.09051).



