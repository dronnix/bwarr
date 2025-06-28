### Very next

- [ ] Check all methods for allocation count and consumed mem (special tests!).
- [ ] Implement special method Init: just split all data to segments and sort it.
- [ ] Implement special method for batch insert: (we know what segments we need to store whole batch, so can use merge segments to infill it).
- [ ] Modernize iterators? 
- [ ] Implement unordered iterator.
- [ ] Write full README;
- [ ] Add GitHub CI for tests and coverage;
- [ ] Release!

- [X]  Profile reusable black: BenchmarkQA_ReplaceOrInsertNotFound-10 case
- [X] Reusable black segment 1/2 of the size of the biggest white segment.
- [X] Implement simple compact method: delete all unused segments.
- [X] Investigate 20 allocations in the benchmark;
- [X] Skip deleted elements during iteration;
- [X] Refactor benchmarks to be able to run iterator benchmarks;
- [X] Make iteration stable in terms of sorting;
- [X] Add other Ascend methods;
- [X] Add backwards iteration;

### Fast release
- [X] Track benchmarks trends, compare with previous commits;
- [X] Add benchmarks to CI;
- [X] Refactor code: move out segment, it's methods and tests;
- [X] Add iterators;

### Full release
- [ ] Plan further steps;
- [ ] Add public benchmarks in separate repository;
- [ ] Benchmark huge structs;

### Compare with competitors
- [X] Find competitor data structures(b-tree, avl-tree, b-plus tree, red-black tree, skip list).
- [X] Add test against B-Tree Generic edition.
- [X] Measure memory consumption.
- [X] Benchmark for 80% deleted case.
- [ ] Add mixed test (insert/find/delete) with specified ratio.
- [ ] Add test against red-black tree.
- [ ] Add test against skip list.

#### Nearest optimization
- [ ] Add pointer-based comparison function;
- [ ] use copy() in demote and merging methods.
- [X] Implement bitset for segment.
- [X] Try to do only one comparison for main data, seq scan or chord method for bits. Benchmark it.
- [X] For each segment add indexes of leftmost and rightmost non-deleted elements.

#### Optimization ideas
- [ ] Make compaction experiment: move all deleted elements to the end of the segment.
- [ ] Invent full compaction: no deleted elements should remain;





 
