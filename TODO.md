### Full release
- [ ] Add public benchmarks in separate repository;
- [ ] Implement special method for batch insert: (we know what segments we need to store whole batch, so can use merge segments to infill it).
- [ ] Modernize iterators?
- [ ] Benchmark huge structs;

### Compare with competitors
- [X] Find competitor data structures(b-tree, avl-tree, b-plus tree, red-black tree, skip list).
- [X] Measure memory consumption.
- [X] Benchmark for 80% deleted case.
- [ ] Add mixed test (insert/find/delete) with specified ratio.


#### Optimization
- [ ] Add pointer-based comparison function;
- [ ] use copy() in demote and merging methods.
- [ ] Make compaction experiment: move all deleted elements to the end of the segment.
- [ ] Invent full compaction: no deleted elements should remain;
- [X] Implement bitset for segment.
