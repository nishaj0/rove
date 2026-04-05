.PHONY: help tray run debug build verify tidy test vet fmt clean

BINARY ?= cloud-saver-daemon.exe
CMD_PATH := ./cmd/daemon

help:
	@echo "backend make targets:"
	@echo "  make tray    - run daemon with tray"
	@echo "  make run     - run daemon without tray"
	@echo "  make debug   - run daemon without tray with debug logs"
	@echo "  make build   - build daemon executable"
	@echo "  make verify  - compile all packages"
	@echo "  make tidy    - tidy modules"
	@echo "  make test    - run tests"
	@echo "  make vet     - run go vet"
	@echo "  make fmt     - run go fmt"
	@echo "  make clean   - remove built daemon executable"

tray:
	go run $(CMD_PATH)

run:
	go run $(CMD_PATH) --no-tray

debug:
	go run $(CMD_PATH) --no-tray --debug

build:
	go build -o $(BINARY) $(CMD_PATH)

verify:
	go build ./...

tidy:
	go mod tidy

test:
	go test ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

clean:
	$(RM) $(BINARY)
