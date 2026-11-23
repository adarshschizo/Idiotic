.PHONY: build test vet clean

build:
	mkdir -p bin
	go build -o bin/idiotic ./cmd/idiotic

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf bin
