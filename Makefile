.PHONY: build vet test install build-opsfleet vet-opsfleet clean
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

# 清理本地构建产物（不删除站点配置与密钥）
clean:
	rm -f $(BINARY)
	rm -f ft-backend/opsfleet-backend
	rm -f ft-client/ft-client
	rm -rf bin dist
	rm -rf ft-front/dist
	find . -name '*.swp' -delete 2>/dev/null || true
