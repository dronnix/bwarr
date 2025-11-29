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
- Easily serializable (work in progress);

### Tradeoffs
- One per *N* insert operations complexity falls down to *O(N)*, amortized remains *O(log N)*. For real-time systems, it may introduce latency spikes for collections with millions of elements.
- For some rare cases with deleting special series of elements, `Search()` operations may degrade to *O(N)/4*. Can be prevented by calling Compact();

###  Benchmarks
TBD

## Usage

### Basic Example

```go
package main

import (
    "fmt"

    "github.com/dronnix/bwarr"
)

func main() {
    // Create a BWArr with an integer comparison function
    // The second parameter (10) is the initial capacity hint
    bwa := bwarr.New(func(a, b int64) int {
        return int(a - b)
    }, 10)

    // Insert elements
    bwa.Insert(42)
    bwa.Insert(17)
    bwa.Insert(99)
    bwa.Insert(23)
    bwa.Insert(8)

    fmt.Printf("Length: %d\n", bwa.Len()) // Output: Length: 5

    // Get an element
    val, found := bwa.Get(42)
    if found {
        fmt.Printf("Found: %d\n", val) // Output: Found: 42
    }

    // Delete an element
    deleted, found := bwa.Delete(17)
    if found {
        fmt.Printf("Deleted: %d\n", deleted) // Output: Deleted: 17
    }

    // Iterate in ascending order
    fmt.Println("Elements in sorted order:")
    bwa.Ascend(func(item int64) bool {
        fmt.Printf("  %d\n", item)
        return true // return false to stop iteration early
    })
    // Output:
    //   8
    //   23
    //   42
    //   99
}
```
## Data Structure
The idea of Black-White Array was invented and published by [Z. George Mou](https://www.cs.brandeis.edu/~mou/) in [Black-White Array: A New Data Structure for
Dynamic Data Sets](https://arxiv.org/abs/2004.09051).


