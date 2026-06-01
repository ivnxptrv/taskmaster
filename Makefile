.PHONY: all build run test test-race vet clean

BIN := taskmaster
CONF := ./configs/config-subj.yaml

all: build

build:
	go build -o $(BIN) ./cmd/taskmaster

run: build
	./$(BIN) -c $(CONF)

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

clean:
	rm -f $(BIN)
