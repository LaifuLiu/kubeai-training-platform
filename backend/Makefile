.PHONY: build fmt run test

fmt:
	gofmt -w main.go

build: fmt
	go build -o bin/kubeai-api .

test:
	go test ./...

run: build
	./bin/kubeai-api