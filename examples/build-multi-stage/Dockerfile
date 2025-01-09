FROM mcr.microsoft.com/devcontainers/go:1.22-bullseye AS go

ARG TARGETOS
ARG TARGETARCH

# Install Node.js
RUN \
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
    apt-get update \
    && apt-get install -y --no-install-recommends protobuf-compiler \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

FROM go AS final

COPY app /app

RUN echo hello
