# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is **bwarr** - a Black-White Array implementation in Go. It's a high-performance ordered data structure that maintains elements in sorted order using a unique segmented architecture with white (sorted) segments and black (temporary) segments.

## Common Development Commands

### Testing and Quality Assurance
- `make test` - Run all tests once
- `go test -count 1 ./...` - Run tests directly
- `make lint` - Run golangci-lint with project configuration
- `golangci-lint run -c qa/.golangci.yml` - Run linter directly
- `make coverage` - Generate test coverage report and open in browser

### Benchmarking
- `make bench` - Run comprehensive benchmarks and performance checks
- `make bench_save` - Save current benchmark results as baseline
- `make bench_show` - Start web server to visualize benchmark trends
- `go test -bench=BenchmarkQA -benchtime 1000000x ./...` - Run quick benchmarks
- `go test -bench=BenchmarkLongQA -benchtime 2s ./...` - Run long-running benchmarks

### Build and Development
- `make all` - Run tests, linting, and benchmarks (full quality check)
- Go version: 1.24 (check go.mod for exact version)

## Architecture Overview

The BWArr data structure implements a novel approach to maintaining sorted collections:

### Core Components

1. **BWArr[T]** (`bwarr.go`): Main data structure containing:
   - `whiteSegments`: Array of sorted segments with exponentially increasing sizes (powers of 2)
   - `highBlackSeg` & `lowBlackSeg`: Reusable temporary segments for merge operations
   - `total`: Bit field tracking which segments contain data
   - `cmp`: Custom comparison function

2. **segment[T]** (`segment.go`): Individual storage segments with:
   - `elements`: Actual data storage
   - `deleted`: Boolean array tracking deleted elements
   - Efficient min/max tracking and binary search capabilities

3. **iterator[T]** (`iterator.go`): Multi-segment iterator supporting:
   - Forward/backward traversal
   - Range queries (greater-than, less-than, between values)
   - Stable sorting behavior

### Key Algorithmic Concepts

- **Segment Hierarchy**: Uses exponentially sized segments (1, 2, 4, 8, 16, ... elements)
- **Lazy Deletion**: Elements are marked as deleted rather than immediately removed
- **Merge-Based Operations**: New elements trigger merging of segments when capacity is exceeded
- **Bit Field Tracking**: Uses `total` field bits to track which segments are active

### Performance Characteristics

The implementation is heavily optimized with:
- Benchmark tracking using gobenchdata for performance regression detection
- Memory-efficient lazy deletion strategy
- Optimized iterators with sorted merge behavior
- Comprehensive linting rules (see qa/.golangci.yml)

## Testing Approach

- Uses testify/assert for test assertions
- Separate benchmark tests for performance monitoring
- QA directory contains benchmark data and tooling
- Coverage reporting with HTML output
- Parallel test execution supported
- Use testify in tests

## Dependencies

- `github.com/google/btree v1.1.3` - Used for performance comparisons
- `github.com/stretchr/testify v1.9.0` - Testing framework
- gobenchdata - Benchmark trend analysis (external tool)