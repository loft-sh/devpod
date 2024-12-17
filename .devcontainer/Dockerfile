FROM mcr.microsoft.com/devcontainers/go:1.22-bullseye

ARG TARGETOS
ARG TARGETARCH

# Install Node.js
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl ca-certificates gnupg \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get update \
    && apt-get install -y --no-install-recommends nodejs \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Set environment variables for Rust
ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:$PATH \
    RUST_VERSION=1.69.0

# Install Rust
RUN set -eux; \
    dpkgArch="$(dpkg --print-architecture)" || exit 1; \
    if [ "${dpkgArch}" = "amd64" ]; then \
        rustArch="x86_64-unknown-linux-gnu"; rustupSha256="0b2f6c8f85a3d02fde2efc0ced4657869d73fccfce59defb4e8d29233116e6db"; \
    elif [ "${dpkgArch}" = "armhf" ]; then \
        rustArch="armv7-unknown-linux-gnueabihf"; rustupSha256="f21c44b01678c645d8fbba1e55e4180a01ac5af2d38bcbd14aa665e0d96ed69a"; \
    elif [ "${dpkgArch}" = "arm64" ]; then \
        rustArch="aarch64-unknown-linux-gnu"; rustupSha256="673e336c81c65e6b16dcdede33f4cc9ed0f08bde1dbe7a935f113605292dc800"; \
    elif [ "${dpkgArch}" = "i386" ]; then \
        rustArch="i686-unknown-linux-gnu"; rustupSha256="e7b0f47557c1afcd86939b118cbcf7fb95a5d1d917bdd355157b63ca00fc4333"; \
    else \
        echo "unsupported architecture: ${dpkgArch}" >&2; exit 1; \
    fi; \
    wget "https://static.rust-lang.org/rustup/archive/1.26.0/${rustArch}/rustup-init" -O /tmp/rustup-init; \
    echo "${rustupSha256} */tmp/rustup-init" | sha256sum -c -; \
    chmod +x /tmp/rustup-init; \
    /tmp/rustup-init -y --no-modify-path --profile minimal --default-toolchain "$RUST_VERSION" --default-host "$rustArch"; \
    rm /tmp/rustup-init; \
    chmod -R a+w "$RUSTUP_HOME" "$CARGO_HOME"; \
    rustup --version; \
    cargo --version; \
    rustc --version

# Install Protobuf compiler
RUN apt-get update \
    && apt-get install -y --no-install-recommends protobuf-compiler \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
