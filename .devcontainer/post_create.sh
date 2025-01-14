
#!/usr/bin/env bash

set -euo pipefail

log() {
  echo "[POST_CREATE] $*"
}

# Start docker daemon. The script should've been put here by the DinD devcontainer feature
log "Starting Docker Daemon"
sudo /usr/local/share/docker-init.sh

REBUILD_SCRIPT="./hack/rebuild.sh"

if [[ ! -f "$REBUILD_SCRIPT" ]]; then
  log "Error: Rebuild script not found at $REBUILD_SCRIPT" >&2
  exit 1
fi

log "Building initial version of devpod binary"
chmod +x "$REBUILD_SCRIPT"
BUILD_PLATFORMS="linux" "$REBUILD_SCRIPT"

# Add our user to docker group 
sudo usermod -aG docker $USER

log "Installing docker provider with default options"
devpod provider add docker

log "Done"
