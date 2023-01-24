#!/bin/sh
set -e

command_exists() {
  command -v "$@" > /dev/null 2>&1
}

user="$(id -un 2>/dev/null || true)"
sh_c='sh -c'
if [ "$user" != 'root' ]; then
  if command_exists sudo; then
    sh_c='sudo -E sh -c'
  elif command_exists su; then
    sh_c='su -c'
  else
    echo Error: this installer needs the ability to run commands as root.
    echo We are unable to find either "sudo" or "su" available to make this happen.
    exit 1
  fi
fi

# Install Docker
if ! command_exists docker; then
  echo "Install Docker"
  if command_exists wget; then
     wget -qO- https://get.docker.com/ | sh
  elif command_exists curl; then
     curl -L https://get.docker.com/ | sh
  else
     echo Error: this installer needs the ability to run commands as root.
     echo We are unable to find either "wget" or "curl" available to make this happen.
     exit 1
  fi
fi

# Install Node
if ! command_exists node; then
  echo "Install Node"
  if command_exists wget; then
    wget -qO- https://deb.nodesource.com/setup_16.x | $sh_c "bash -" && $sh_c "apt-get install -y nodejs"
  elif command_exists curl; then
    curl -fsSL https://deb.nodesource.com/setup_16.x | $sh_c "bash -" && $sh_c "apt-get install -y nodejs"
  else
    echo Error: this installer needs the ability to run commands as root.
    echo We are unable to find either "wget" or "curl" available to make this happen.
    exit 1
  fi
fi

# Install DevContainer
if ! command_exists devcontainer; then
  echo "Install DevContainer"
  # Install build credentials
  $sh_c "apt-get install -y build-essential git"

  # Install devcontainer cli
  $sh_c "npm install -g @devcontainers/cli"
fi