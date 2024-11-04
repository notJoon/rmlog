MAIN_PACKAGE=./cmd/rmlog

all: test install

install:
	go install $(MAIN_PACKAGE)

test:
	go test ./...

.PHONY: install test