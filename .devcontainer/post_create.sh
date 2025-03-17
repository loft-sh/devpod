#!/usr/bin/env bash

set -euo pipefail

log() {
  echo "[POST_CREATE] $*"
}

# Start docker daemon. The script should've been put here by the DinD devcontainer feature
log "Starting Docker Daemon"
sudo /usr/local/share/docker-init.sh

# Add devpod user to docker group
sudo usermod -aG docker devpod

# Ensure the .devpod directory is owned by devpod user
sudo chown -R devpod:devpod /home/devpod/.devpod

log "Installing docker provider with default options as devpod user"
sudo -u devpod devpod provider add docker

log "Done"
