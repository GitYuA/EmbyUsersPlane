FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/emby-users-panel ./main.go

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Shanghai
ENV APP_DATA_DIR=/data

WORKDIR /app

COPY --from=builder /out/emby-users-panel /usr/local/bin/emby-users-panel
COPY public ./public
COPY templates ./templates

RUN mkdir -p /data/log /data/users /data/rate_limit

EXPOSE 8086 8085

ENTRYPOINT ["/usr/local/bin/emby-users-panel"]
