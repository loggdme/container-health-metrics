# 🐳 Container Health Metrics

A lightweight Go service that monitors Docker container health status and exposes metrics via a REST API.

## ✨ Features

- 🪶 Extremely lightweight
- 🔍 Monitors all Docker containers on the host
- 📊 Exposes container states (healthy, unhealthy, running, exited)
- 🚀 Simple REST API with rate limiting
- 🐋 Containerized and ready to deploy

## 🚀 Quick Start

### Using Docker

```bash
docker build -t container-health-metrics .
docker run -d \
  -p 9066:9066 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  container-health-metrics
```

**Note:** The container runs as root to ensure Docker socket access works on both Linux and macOS. On macOS (Docker Desktop/OrbStack), the socket is typically root-owned inside containers, so root access is required. On Linux systems, you can run as a non-root user by mounting the socket with appropriate permissions and using `--user` flag if needed.

### Environment Variables

- `PORT` - Server port (default: `9066`)

## 📡 API Endpoints

### `GET /metrics`

Returns the health status of all Docker containers.

**Response:**
```json
{
  "container1": "healthy",
  "container2": "running",
  "container3": "unhealthy"
}
```

### `GET /health`

Health check endpoint for the service itself.

**Response:**
```json
{
  "status": "ok"
}
```

## ©️ License

This project and each package it provides is licensed under the MIT License - see the [LICENSE](LICENSE) file for more details.
