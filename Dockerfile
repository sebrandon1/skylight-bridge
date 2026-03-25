FROM golang:1.26-alpine AS builder

ARG VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-X main.Version=${VERSION}" -o /skylight-bridge .

FROM alpine:3

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1000 bridge && \
    mkdir -p /data /config && \
    chown bridge:bridge /data

COPY --from=builder /skylight-bridge /usr/local/bin/skylight-bridge

USER bridge
EXPOSE 8080
VOLUME ["/data", "/config"]

ENTRYPOINT ["skylight-bridge"]
CMD ["--config", "/config/config.yaml"]
