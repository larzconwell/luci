test:
	go test ./...

testrace test-race:
	go test -race ./...

lint:
	golangci-lint run
