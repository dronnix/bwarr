all: test lint bench

# Compare serial QA benchmarks independently of the host's GOMAXPROCS suffix.
NORMALIZE_BENCH_NAMES = sed -E 's/("Name"[[:space:]]*:[[:space:]]*"Benchmark[^"]*)-[0-9]+"/\1"/g'

bench eb: SHELL := /bin/bash -o pipefail

test:
	go test -count 1 ./...

lint:
	golangci-lint run -c qa/.golangci.yml

coverage:
	go test -v ./... -coverprofile qa/cover.tmp
	go tool cover -html qa/cover.tmp -o qa/coverage.html
	rm qa/cover.tmp

coverage_show: coverage
	open qa/coverage.html

bench:
	go test -bench=BenchmarkQA -benchtime 1000000x ./... | tee qa/bench.tmp
	go test -bench=BenchmarkLongQA -benchtime 2s ./... | tee -a qa/bench.tmp
	cat qa/bench.tmp | gobenchdata --require-benchmarks --json qa/cur_bench.json
	gobenchdata checks eval --checks.config qa/gobenchdata-checks.yml <($(NORMALIZE_BENCH_NAMES) qa/last_bench.json) <($(NORMALIZE_BENCH_NAMES) qa/cur_bench.json) --json qa/rep.json
	gobenchdata checks --checks.config qa/gobenchdata-checks.yml report qa/rep.json

eb:
	gobenchdata checks eval --checks.config qa/gobenchdata-checks.yml <($(NORMALIZE_BENCH_NAMES) qa/dmitry_last_bench.json) <($(NORMALIZE_BENCH_NAMES) qa/cur_bench.json) --json qa/rep.json
	gobenchdata checks --checks.config qa/gobenchdata-checks.yml report qa/rep.json

bench_save:
	cat qa/bench.tmp |  gobenchdata --append --json qa/all_bench.json
	cp qa/cur_bench.json qa/last_bench.json

bench_show:
	mkdir -p qa/web_tmp
	gobenchdata web generate qa/web_tmp
	cp qa/all_bench.json qa/web_tmp/benchmarks.json
	cd qa/web_tmp && gobenchdata web serve
