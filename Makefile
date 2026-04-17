.PHONY: build build-executor vet test install build-opsfleet vet-opsfleet clean
BINARY=ai-sre
EXECUTOR=bin/opsfleet-executor

build:
	go build -o $(BINARY) .

# 与 ai-sre 同引擎，部署在受管机上的 OpsFleet 本地执行器
build-executor:
	mkdir -p bin
	go build -trimpath -ldflags="-s -w" -o $(EXECUTOR) ./cmd/opsfleet-executor

vet:
	go vet ./...

test:
	go test ./...

install: build
	install -m 0755 $(BINARY) /usr/local/bin/$(BINARY) 2>/dev/null || cp $(BINARY) ~/bin/$(BINARY)

# OpsFleetPilot（Vue3 + Gin），与 CLI 同仓；需 Node/npm 才能完整构建前端
build-opsfleet:
	bash scripts/build-all.sh

vet-opsfleet:
	cd ft-backend && go vet ./...

# 清理本地构建产物（不删除站点配置与密钥）
clean:
	rm -f $(BINARY)
	rm -f $(EXECUTOR)
	rm -f ft-backend/opsfleet-backend
	rm -rf bin dist
	rm -rf ft-front/dist
	find . -name '*.swp' -delete 2>/dev/null || true
