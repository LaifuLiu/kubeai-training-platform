.PHONY: build fmt run

fmt:
	gofmt -w main.go

build: fmt
	go build -o bin/kubeai-api .

run: build
	./bin/kubeai-api