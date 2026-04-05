# STAGE: Go
FROM golang:1.23-bookworm AS go-builder
ENV CGO_ENABLED=1 \
    GO111MODULE=on
WORKDIR /gee
# Dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    gcc libc6-dev libsqlite3-dev
COPY . .
RUN go mod download && \
    go run scripts/init/main.go -copyData="config.json,database.db,index.tmpl" && \
    go build -o ./dist/api ./cmd/api/main.go

# STAGE: Base
FROM debian:bookworm-slim as base-builder
ENV TZ=Asia/Shanghai
WORKDIR /web
# Dependencies
RUN apt-get update && \
    apt-get install -y curl vim tzdata gnupg ca-certificates dos2unix && \
    update-ca-certificates && \
    rm -rf /var/lib/apt/lists/*
# Go Service
COPY --from=go-builder /gee/dist/api /web/api
COPY --from=go-builder /gee/data /web/data
# Entrypoint Script
COPY ./scripts/docker-entrypoint.sh /web/docker-entrypoint.sh
RUN chmod +x /web/api && \
    dos2unix /web/docker-entrypoint.sh && \
    chmod +x /web/docker-entrypoint.sh
EXPOSE 3000
ENTRYPOINT ["/web/docker-entrypoint.sh"]
CMD ["/web/api", "--config-path=/web/data/config.json"]
