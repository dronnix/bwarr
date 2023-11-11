all: test lint

test:
	go test ./...

lint:
	golangci-lint run

coverage:
	go test -v ./... -coverprofile cover.out
	go tool cover -html cover.out -o coverage.html
	rm cover.out

