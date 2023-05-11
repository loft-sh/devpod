FROM mcr.microsoft.com/devcontainers/go:0-1.19-bullseye

ARG TARGETOS
ARG TARGETARCH

RUN curl -fsSL https://deb.nodesource.com/setup_19.x | bash - && \
    apt-get update && \
    apt-get install -y nodejs
