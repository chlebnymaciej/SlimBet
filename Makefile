.PHONY: build run test lint clean

build:
	go build -o ./bin/server .

run: build
	./bin/server

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf ./bin/
