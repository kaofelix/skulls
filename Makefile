.PHONY: build test lint clean install

BINARY := skulls
CMD := ./cmd/skulls

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY)

install:
	go install $(CMD)
