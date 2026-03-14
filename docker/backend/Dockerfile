# ---- 构建阶段 ----
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /app/etcd-manager ./cmd/server

# ---- 运行阶段 ----
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata sqlite-libs

WORKDIR /app

COPY --from=builder /app/etcd-manager .
COPY configs/config.yaml ./configs/config.yaml

RUN mkdir -p /app/data

ENV CONFIG_PATH=/app/configs/config.yaml
ENV TZ=Asia/Shanghai

EXPOSE 8080

ENTRYPOINT ["./etcd-manager"]
