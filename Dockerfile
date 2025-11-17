FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum* ./
RUN go mod download

COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o docker-health-exporter .


FROM alpine:3.22.2

COPY --from=builder /build/docker-health-exporter /usr/local/bin/docker-health-exporter

EXPOSE 9066

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["wget", "-qO-", "http://localhost:9066/health"]
  
ENTRYPOINT ["docker-health-exporter"]
