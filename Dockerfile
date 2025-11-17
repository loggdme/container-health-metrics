FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum* ./
RUN go mod download

COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o docker-health-exporter .


FROM alpine:3.20

RUN apk --no-cache add ca-certificates docker-cli wget
RUN adduser -D -g '' deploy

COPY --from=builder /build/docker-health-exporter /usr/local/bin/docker-health-exporter

USER deploy
EXPOSE 9066

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:9066/health || exit 1

ENTRYPOINT ["docker-health-exporter"]

