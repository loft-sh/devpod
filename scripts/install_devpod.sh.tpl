#!/bin/sh
set -e

command_exists() {
  command -v "$@" > /dev/null 2>&1
}

is_arm() {
  case "$(uname -a)" in
  *arm* ) true;;
  *arm64* ) true;;
  *aarch* ) true;;
  *aarch64* ) true;;
  * ) false;;
  esac
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

DEVPOD_PATH={{ .AgentPath }}
if [ ! -f "$DEVPOD_PATH" ]; then
  DOWNLOAD_URL={{ .BaseUrl }}/devpod-linux-amd64
  if is_arm; then
    DOWNLOAD_URL={{ .BaseUrl }}/devpod-linux-arm64
  fi

  while :; do
    status=""
    if command_exists curl; then
        $sh_c "curl -fsSL $DOWNLOAD_URL -o $DEVPOD_PATH" && break
        status=$?
    elif command_exists wget; then
        $sh_c "wget -q $DOWNLOAD_URL -O $DEVPOD_PATH" && break
        status=$?
    else
        echo "error: no download tool found, please install curl or wget"
        exit 127
    fi
    echo "error: failed to download devpod"
    echo "       command returned: ${status}"
    echo "Trying again in 10 seconds..."
    sleep 10
  done

  if ! $sh_c "chmod +x $DEVPOD_PATH"; then
      echo "Failed to make $DEVPOD_PATH executable"
      exit 1
  fi
fi

{{ if .Command }}
{{ .Command }}
{{ else if .Token }}
exec $DEVPOD_PATH helper ssh-server --token "{{ .Token }}"
{{ end }}