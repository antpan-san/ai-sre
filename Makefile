.PHONY: build vet test install build-opsfleet vet-opsfleet
BINARY=ai-sre

build:
	go build -o $(BINARY) .

vet:
	go vet ./...

test:
	go test ./...

install: build
	install -m 0755 $(BINARY) /usr/local/bin/$(BINARY) 2>/dev/null || cp $(BINARY) ~/bin/$(BINARY)

# OpsFleetPilot（Vue3 + Gin + Agent），与 CLI 同仓；需 Node/npm 才能完整构建前端
build-opsfleet:
	bash scripts/build-all.sh

vet-opsfleet:
	cd ft-backend && go vet ./...
