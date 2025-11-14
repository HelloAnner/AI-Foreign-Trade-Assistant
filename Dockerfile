# Unified builder stage (installs Node & Go manually to avoid extra base pulls)
FROM debian:bookworm-slim AS builder

ARG NODE_VERSION=22.12.0
ARG GO_VERSION=1.22.6
ARG GOPROXY=https://goproxy.cn,direct
ARG TARGETARCH
ENV GOPROXY=${GOPROXY}
ENV PATH=/usr/local/go/bin:/usr/local/bin:$PATH
ENV PLAYWRIGHT_DRIVER_PATH=/out/playwright/driver \
    PLAYWRIGHT_BROWSERS_PATH=/out/playwright/browsers
ENV DEBIAN_FRONTEND=noninteractive

RUN printf 'deb http://mirrors.aliyun.com/debian/ bookworm main contrib non-free non-free-firmware\n' > /etc/apt/sources.list \
    && printf 'deb http://mirrors.aliyun.com/debian/ bookworm-updates main contrib non-free non-free-firmware\n' >> /etc/apt/sources.list \
    && printf 'deb http://mirrors.aliyun.com/debian-security bookworm-security main contrib non-free non-free-firmware\n' >> /etc/apt/sources.list \
    && rm -f /etc/apt/sources.list.d/debian.sources

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        git \
        build-essential \
        xz-utils \
        wget \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js (architecture aware)
RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) NODE_DIST="linux-x64" ;; \
      arm64) NODE_DIST="linux-arm64" ;; \
      *) echo "unsupported TARGETARCH=${TARGETARCH} for Node"; exit 1 ;; \
    esac; \
    curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-${NODE_DIST}.tar.xz" \
      | tar -xJ -C /usr/local --strip-components=1

# Install Go toolchain (architecture aware)
RUN set -eux; \
    rm -rf /usr/local/go; \
    case "${TARGETARCH}" in \
      amd64) GO_DIST="linux-amd64" ;; \
      arm64) GO_DIST="linux-arm64" ;; \
      *) echo "unsupported TARGETARCH=${TARGETARCH} for Go"; exit 1 ;; \
    esac; \
    curl -fsSL "https://dl.google.com/go/go${GO_VERSION}.${GO_DIST}.tar.gz" \
      | tar -xz -C /usr/local

# Build frontend assets
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build backend binary
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN rm -rf ./static && mkdir -p ./static \
    && cp -r /src/frontend/dist/. ./static
RUN CGO_ENABLED=1 GOOS=linux GOARCH=${TARGETARCH} go build -o /out/ai-trade-assistant main.go

# Pre-install Playwright driver and browsers
RUN mkdir -p "$PLAYWRIGHT_DRIVER_PATH" "$PLAYWRIGHT_BROWSERS_PATH" \
    && go run github.com/playwright-community/playwright-go/cmd/playwright@v0.4902.0 install chromium

# Runtime stage
FROM debian:bookworm-slim

ENV PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=true \
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 \
    PLAYWRIGHT_BROWSERS_PATH=/opt/playwright/browsers \
    PLAYWRIGHT_DRIVER_PATH=/opt/playwright/driver \
    HOME=/data \
    FOREIGN_TRADE_NO_BROWSER=1 \
    DEBIAN_FRONTEND=noninteractive

RUN printf 'deb http://mirrors.aliyun.com/debian/ bookworm main contrib non-free non-free-firmware\n' > /etc/apt/sources.list \
    && printf 'deb http://mirrors.aliyun.com/debian/ bookworm-updates main contrib non-free non-free-firmware\n' >> /etc/apt/sources.list \
    && printf 'deb http://mirrors.aliyun.com/debian-security bookworm-security main contrib non-free non-free-firmware\n' >> /etc/apt/sources.list \
    && rm -f /etc/apt/sources.list.d/debian.sources

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        fonts-liberation \
        libasound2 \
        libatk-bridge2.0-0 \
        libatk1.0-0 \
        libcups2 \
        libdrm2 \
        libgbm1 \
        libglib2.0-0 \
        libgtk-3-0 \
        libnspr4 \
        libnss3 \
        libx11-6 \
        libx11-xcb1 \
        libxcb1 \
        libxcomposite1 \
        libxdamage1 \
        libxext6 \
        libxfixes3 \
        libxrandr2 \
        libxshmfence1 \
        wget \
        xdg-utils \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/ai-trade-assistant /app/ai-trade-assistant
COPY --from=builder /out/playwright /opt/playwright
COPY docker/entrypoint.sh /usr/local/bin/ai-entrypoint.sh
RUN chmod +x /usr/local/bin/ai-entrypoint.sh

VOLUME ["/data"]
EXPOSE 7860

ENTRYPOINT ["ai-entrypoint.sh"]
CMD ["/app/ai-trade-assistant"]
