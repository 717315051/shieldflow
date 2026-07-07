# ============================================================================
# ShieldFlow Dockerfile (可选 Docker 支持)
# 多阶段构建: golang:1.22 编译 → node:18 前端 → debian:slim 运行
# 构建镜像标签: shieldflow/shieldflow:latest
# 用法: docker build -t shieldflow/shieldflow:latest .
# ============================================================================
FROM golang:1.22 AS go-builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/backend     ./cmd/backend      && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/grpc-server ./cmd/grpc-server  && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/edge        ./cmd/edge         && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/dns-sync    ./cmd/dns-sync     && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/log-server  ./cmd/log-server

FROM node:18 AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm install --silent 2>/dev/null || npm install
COPY web/ .
RUN npm run build

FROM debian:bookworm-slim AS runtime
LABEL org.opencontainers.image.title="ShieldFlow"
LABEL org.opencontainers.image.description="ShieldFlow 企业级自建CDN系统"
LABEL org.opencontainers.image.source="https://github.com/shieldflow/shieldflow"

RUN apt-get update -qq && \
    apt-get install -y --no-install-recommends \
        ca-certificates supervisor nginx openssl && \
    rm -rf /var/lib/apt/lists/*

# 复制二进制
COPY --from=go-builder /out/backend     /usr/local/shieldflow/bin/backend
COPY --from=go-builder /out/grpc-server /usr/local/shieldflow/bin/grpc-server
COPY --from=go-builder /out/edge        /usr/local/shieldflow/bin/edge
COPY --from=go-builder /out/dns-sync    /usr/local/shieldflow/bin/dns-sync
COPY --from=go-builder /out/log-server  /usr/local/shieldflow/bin/log-server

# 复制前端
COPY --from=web-builder /web/dist /usr/local/shieldflow/web

# 复制配置模板
COPY deploy/ /opt/shieldflow/deploy/

# 运行目录
RUN mkdir -p /etc/shieldflow/certs /var/log/shieldflow /var/cache/shieldflow /var/www/acme

# 默认暴露端口
# 80/443: Edge/Nginx  8080: Backend API  50051: gRPC  9527: 健康检查
EXPOSE 80 443 8080 50051 9527 9000 5432 6379

# 入口脚本
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["master"]
