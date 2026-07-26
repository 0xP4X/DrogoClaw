# ═══════════════════════════════════════════════════════════
# 🐉🔥 DrogonClaw — Multi-stage Go Build
# ═══════════════════════════════════════════════════════════

# Stage 1: Build the Go binary
FROM golang:1.26-bookworm AS builder

WORKDIR /build

# Cache Go module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /build/drogonclaw ./cmd/drogonclaw/

# Stage 2: Runtime image with pentesting tools
FROM kalilinux/kali-rolling

# Install core pentesting tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    nmap masscan amass gobuster ffuf sqlmap nuclei \
    nikto dirb whatweb hydra john hashcat \
    metasploit-framework exploitdb \
    python3 python3-pip gcc git curl wget \
    docker.io \
    ca-certificates \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy built binary from builder
COPY --from=builder /build/drogonclaw /app/drogonclaw
COPY --from=builder /build/skills /app/skills
COPY --from=builder /build/skills_manifest.json /app/skills_manifest.json

# Create data directories
RUN mkdir -p /app/data /app/reports

# Create non-root user for the control plane (sandbox itself runs privileged)
RUN useradd -r -m -s /bin/bash drogon
RUN chown -R drogon:drogon /app
USER drogon

# Expose gateway port
EXPOSE 18789

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:18789/health || exit 1

# Start DrogonClaw
ENTRYPOINT ["/app/drogonclaw"]
