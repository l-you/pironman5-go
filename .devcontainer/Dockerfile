FROM mcr.microsoft.com/devcontainers/go:2.1.1-1.26-trixie

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        device-tree-compiler \
        gpiod \
        i2c-tools \
        kmod \
        libgpiod-dev \
        lsof \
    && mkdir -p /go/pkg/mod /home/vscode/.cache/go-build \
    && chown -R vscode:vscode /go /home/vscode/.cache \
    && rm -rf /var/lib/apt/lists/*
