
#!/usr/bin/env bash

set -euo pipefail

log() {
  echo "[POST_CREATE] $*"
}

# Start docker daemon. The script should've been put here by the DinD devcontainer feature
log "Starting Docker Daemon"
sudo /usr/local/share/docker-init.sh
#
# Add our user to docker group 
sudo usermod -aG docker $USER

log "Installing docker provider with default options"
devpod provider add docker

log "Done"
