### Fast release
- [X] Track benchmarks trends, compare with previous commits;
- [X] Add benchmarks to CI;
- [X] Refactor code: move out segment, it's methods and tests;
- [ ] Add iterators;
- [ ] Plan further steps;

### Compare with competitors
- [X] Find competitor data structures(b-tree, avl-tree, b-plus tree, red-black tree, skip list).
- [X] Add test against B-Tree Generic edition.
- [X] Measure memory consumption.
- [X] Benchmark for 80% deleted case.
- [ ] Add mixed test (insert/find/delete) with specified ratio.
- [ ] Add test against red-black tree.
- [ ] Add test against skip list.

#### Nearest optimization
- [X] Implement bitset for segment.
- [X] Try to do only one comparison for main data, seq scan or chord method for bits. Benchmark it.
- [X] For each segment add indexes of leftmost and rightmost non-deleted elements.
- [ ] use copy() in demote and merging methods.
- [ ] Implement special method for batch insert: (we know what segments we need to store whole batch, so can use merge segments to infill it).
- [ ] Implement special method Init: just split all data to segments and sort it.

#### Lack of functionality
- [X] Implement max/min methods.
- [ ] Implement Range methods.

#### Optimization ideas
- [X] Delete unused highest segment (or provide a method).
- [ ] Reusable black segment 1/2 of the size of the biggest white segment.
- [ ] Make compaction experiment: move all deleted elements to the end of the segment.
- [ ] Combo of previous two: method Compact() that get rid of all deleted elements and keep only used segments.
- [ ] Cmp func based on pointers;





 
