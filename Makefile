
BUILD_COMMIT := $(shell git log --format="%H" -n 1)

.PHONY: test
test:
	go test -v 2>&1 |go-junit-report > test_report.xml
	go test -race ./...

.PHONY: lint
lint:
	 golangci-lint run -c golangci-lint.yaml

.PHONY: build
build:
	@echo main.BuildCommit = $(BUILD_COMMIT)
	go build ./...

