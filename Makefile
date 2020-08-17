test:
	go test -cover ./...

race:
	go test -race ./...

lint:
	golangci-lint run --new-from-rev=$(MASTER_COMMIT) ./...

lintci-deps:
	rm -f ./build/bin/golangci-lint
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b ./build/bin v1.30.0
