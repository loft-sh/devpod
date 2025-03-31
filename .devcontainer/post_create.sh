
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
sudo usermod -aG docker devpod
sudo chown devpod:docker /var/run/docker.sock

log "Installing docker provider with default options"
devpod provider add docker


log "Installing vCluster CLI"
curl -L -o vcluster "https://github.com/loft-sh/vcluster/releases/download/v0.24.0/vcluster-linux-amd64" && sudo install -c -m 0755 vcluster /usr/local/bin && rm -f vcluster

log "Done"

log "Installing and starting SSH server for testing"
sudo apt-get update
sudo apt-get install -y openssh-server
sudo service ssh start
