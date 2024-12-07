build:
	CGO_ENABLED=0 GOOS=linux go build -o ass -ldflags "-s -w"
lint:
	golangci-lint run

.PHONY: build lint
