# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS build
WORKDIR /src
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache git ca-certificates \
    && git config --global http.sslVerify true
ENV GOPROXY=https://goproxy.cn,direct \
    GOTOOLCHAIN=local \
    CGO_ENABLED=0 \
    GOOS=linux
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/goreadwise ./cmd/server

FROM alpine:3.21
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -H -u 65532 app
WORKDIR /app
COPY --from=build /out/goreadwise /app/goreadwise
COPY testdata /app/testdata
COPY migrations /app/migrations
ENV TZ=Asia/Shanghai
USER app
EXPOSE 8080
HEALTHCHECK --interval=8s --timeout=4s --retries=8 CMD wget -qO- http://127.0.0.1:8080/api/v1/health || exit 1
ENTRYPOINT ["/app/goreadwise"]
