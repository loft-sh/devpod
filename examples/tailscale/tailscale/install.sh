#!/usr/bin/env bash
# Copyright (c) 2022 Tailscale Inc & AUTHORS All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

set -euo pipefail
apt update
apt install -y curl
curl -fsSL https://tailscale.com/install.sh | sh

script_dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
install "$script_dir/tailscaled-entrypoint.sh" /usr/local/sbin/tailscaled-entrypoint

mkdir -p /var/lib/tailscale /var/run/tailscale

if ! command -v iptables >& /dev/null; then
  if command -v apt-get >& /dev/null; then
    apt-get update
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends iptables
    rm -rf /var/lib/apt/lists/*
  else
    echo "WARNING: iptables not installed. tailscaled might fail."
  fi
fi

echo done