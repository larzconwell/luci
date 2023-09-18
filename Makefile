test:
	go test -v ./...

test-race:
	go test -race -v ./...

lint:
	golangci-lint run
