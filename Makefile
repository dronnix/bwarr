all: test lint bench

test:
	go test ./...

lint:
	golangci-lint run

coverage:
	go test -v ./... -coverprofile qa/cover.tmp
	go tool cover -html qa/cover.tmp -o qa/coverage.html
	rm qa/cover.tmp

bench:
	go test -bench=BenchmarkQA -benchtime 1000000x ./... | tee qa/bench.tmp
	cat qa/bench.tmp | gobenchdata --json qa/cur_bench.json
	gobenchdata checks eval --checks.config qa/gobenchdata-checks.yml qa/cur_bench.json qa/last_bench.json --json qa/rep.json
	gobenchdata checks --checks.config qa/gobenchdata-checks.yml report qa/rep.json

bench_save:
	cat qa/bench.tmp |  gobenchdata --append --json qa/all_bench.json
	cp qa/cur_bench.json qa/last_bench.json

bench_show:
	mkdir -p qa/web_tmp
	gobenchdata web generate qa/web_tmp
	cp qa/all_bench.json qa/web_tmp/benchmarks.json
	cd qa/web_tmp && gobenchdata web serve
