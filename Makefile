# ============================================================================
# ShieldFlow Makefile 构建入口
# 用法:
#   make build         # 编译所有后端组件
#   make build-edge    # 仅编译 edge
#   make web           # 编译前端
#   make install       # 一键安装（需 root）
#   make deploy        # 部署配置和二进制到目标路径
#   make install-edge  # 安装边缘节点
#   make docker        # 构建 Docker 镜像
#   make docker-up     # docker compose 启动
#   make backup        # 数据备份
#   make restore FILE= # 数据恢复
#   make clean         # 清理构建产物
#   make test          # 运行测试
#   make proto         # 生成 gRPC 代码
# ============================================================================

GO          ?= go
GOPROXY     ?= https://goproxy.cn,direct
GOPATH      ?= $(shell $(GO) env GOPATH)
VERSION     ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
BUILD_TIME  ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

PREFIX      ?= /usr/local/shieldflow
CONF_DIR    ?= /etc/shieldflow
BIN_DIR     := $(PREFIX)/bin

COMPONENTS  := backend grpc-server edge dns-sync log-server

# ---------- 构建 ----------
.PHONY: build $(addprefix build-,$(COMPONENTS)) web all

all: build web

build: $(addprefix build-,$(COMPONENTS))

$(addprefix build-,$(COMPONENTS)): build-%:
	@echo "  编译 $*..."
	@mkdir -p $(BIN_DIR)
	GOPROXY=$(GOPROXY) $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$* ./cmd/$*

web:
	@echo "  编译前端..."
	@cd web && npm install --silent && npm run build
	@rm -rf $(PREFIX)/web && cp -r web/dist $(PREFIX)/web

# ---------- gRPC proto ----------
.PHONY: proto
proto:
	@command -v protoc >/dev/null 2>&1 || { echo "请安装 protoc"; exit 1; }
	@echo "  生成 gRPC 代码..."
	@protoc --go_out=. --go-grpc_out=. --proto_path=proto proto/shieldflow.proto

# ---------- 部署 / 安装 ----------
.PHONY: deploy install install-edge uninstall upgrade

deploy: build web
	@echo "  部署配置模板到 $(CONF_DIR)/..."
	@mkdir -p $(CONF_DIR) $(CONF_DIR)/certs /var/log/shieldflow /var/cache/shieldflow
	@for f in backend.yaml grpc.yaml edge.yaml dns-sync.yaml log-server.yaml; do \
		[ -f deploy/$$f ] && cp -n deploy/$$f $(CONF_DIR)/$$f; \
	done
	@echo "  部署 Supervisor 配置..."
	@mkdir -p /etc/supervisord.d
	@for f in deploy/supervisor/*.conf; do [ -f $$f ] && cp -n $$f /etc/supervisord.d/; done
	@echo "  部署 Nginx 配置..."
	@mkdir -p /etc/nginx/conf.d
	@[ -f deploy/nginx/shieldflow.conf ] && cp -n deploy/nginx/shieldflow.conf /etc/nginx/conf.d/ || true
	@echo "  ✅ 部署完成"

install:
	@echo "  运行一键安装脚本..."
	@sudo bash scripts/install.sh

install-edge:
	@echo "  运行边缘节点安装脚本..."
	@[ -n "$(NODE_ID)" ] || { echo "请指定 NODE_ID，例如: make install-edge NODE_ID=edge-01 MASTER=1.2.3.4:50051"; exit 1; }
	@sudo bash scripts/install-edge.sh --node-id $(NODE_ID) --master $(MASTER)

uninstall:
	@sudo bash scripts/uninstall.sh

upgrade:
	@sudo bash scripts/upgrade.sh

# ---------- 数据备份/恢复 ----------
.PHONY: backup restore

backup:
	@sudo bash scripts/backup.sh

restore:
	@[ -n "$(FILE)" ] || { echo "用法: make restore FILE=/var/backups/shieldflow/xxx.tar.gz"; exit 1; }
	@sudo bash scripts/restore.sh --file $(FILE)

# ---------- Docker ----------
.PHONY: docker docker-build docker-up docker-down docker-logs

docker docker-build:
	@docker build -t shieldflow/shieldflow:latest .

docker-up:
	@docker compose up -d

docker-down:
	@docker compose down

docker-logs:
	@docker compose logs -f

# ---------- 测试 / 清理 ----------
.PHONY: test test-race vet fmt lint clean

test:
	@$(GO) test ./... -v

test-race:
	@$(GO) test ./... -race -v

vet:
	@$(GO) vet ./...

fmt:
	@$(GO) fmt ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint 未安装，跳过"

clean:
	@echo "  清理构建产物..."
	@rm -rf $(BIN_DIR) web/dist
	@$(GO) clean -cache 2>/dev/null || true

# ---------- 帮助 ----------
.PHONY: help
help:
	@echo "ShieldFlow Makefile v$(VERSION)"
	@echo ""
	@echo "常用目标:"
	@echo "  make build          编译所有后端组件"
	@echo "  make web            编译前端"
	@echo "  make all            编译后端+前端"
	@echo "  make deploy         部署到本机 (PREFIX=$(PREFIX))"
	@echo "  make install        一键安装"
	@echo "  make install-edge   安装边缘节点 (NODE_ID=... MASTER=...)"
	@echo "  make upgrade        升级"
	@echo "  make docker         构建 Docker 镜像"
	@echo "  make docker-up      docker compose 启动"
	@echo "  make backup         数据备份"
	@echo "  make restore FILE=x 数据恢复"
	@echo "  make test           运行测试"
	@echo "  make proto          生成 gRPC 代码"
	@echo "  make clean          清理"
