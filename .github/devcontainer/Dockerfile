FROM mcr.microsoft.com/devcontainers/go:1.23-bullseye

ARG TARGETOS
ARG TARGETARCH

# We want to setup our own user later, this removes the built-in VSCode user 
# that comes with the base image
RUN sudo userdel -r vscode -f || true && rm -rf /home/vscode

# Install Node.js
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl ca-certificates gnupg \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get update \
    && apt-get install -y --no-install-recommends nodejs \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Protobuf compiler
RUN apt-get update \
    && apt-get install -y --no-install-recommends protobuf-compiler \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install kind 
RUN curl -L -o kind "https://kind.sigs.k8s.io/dl/v0.17.0/kind-linux-${TARGETARCH}" && install -c -m 0755 kind /usr/local/bin && rm kind

# Install alacritty terminfo
RUN wget https://raw.githubusercontent.com/alacritty/alacritty/master/extra/alacritty.info \
    && sudo tic -xe alacritty,alacritty-direct alacritty.info \
    && rm alacritty.info
