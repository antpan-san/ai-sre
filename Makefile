.PHONY: build vet test install
BINARY=ai-sre

build:
	go build -o $(BINARY) .

vet:
	go vet ./...

test:
	go test ./...

install: build
	install -m 0755 $(BINARY) /usr/local/bin/$(BINARY) 2>/dev/null || cp $(BINARY) ~/bin/$(BINARY)
