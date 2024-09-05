FROM mcr.microsoft.com/devcontainers/go:1.22-bullseye

ARG TARGETOS
ARG TARGETARCH

# Install Node.js
RUN \
    --mount=type=cache,target=/var/cache/apt \
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get update \
    && apt-get install -y --no-install-recommends nodejs \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Set environment variables for Rust
ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:$PATH \
    RUST_VERSION=1.69.0

# Install Protobuf compiler
RUN \
    --mount=type=cache,target=/var/cache/apt \
    apt-get update \
    && apt-get install -y --no-install-recommends protobuf-compiler \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

COPY app /app
COPY files /files

RUN echo hello