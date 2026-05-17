.PHONY: fmt lint test vuln clean run

fmt:
	gofumpt -w .

lint:
	golangci-lint run

test:
	go test ./... -cover -race -v

vuln:
	govulncheck ./...

run:
	go run ./cmd/api

clean:
	go clean
