test:
	go test ./...

test-race:
	go test -race ./...

lint:
	golangci-lint run
