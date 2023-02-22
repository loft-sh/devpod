#!/bin/sh
set -e

INSTALL_DIR="{{ .InstallDir }}"
INSTALL_FILENAME="{{ .InstallFilename }}"

INSTALL_PATH="$INSTALL_DIR/$INSTALL_FILENAME"
PREFER_DOWNLOAD="{{ .PreferDownload }}"
CHMOD_PATH="{{ .ChmodPath }}"

# start marker
echo "start"

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

is_arm() {
  case "$(uname -a)" in
  *arm* ) true;;
  *arm64* ) true;;
  *aarch* ) true;;
  *aarch64* ) true;;
  * ) false;;
  esac
}

DOWNLOAD_URL={{ .DownloadAmd }}
if is_arm; then
  DOWNLOAD_URL={{ .DownloadArm }}
fi

inject() {
  echo "ARM-$(is_arm && echo -n 'true' || echo -n 'false')"
  cat > $INSTALL_PATH
}

download() {
  while :; do
    status=""
    if command_exists curl; then
        $sh_c "curl -fsSL $DOWNLOAD_URL -o $INSTALL_PATH" && break
        status=$?
    elif command_exists wget; then
        $sh_c "wget -q $DOWNLOAD_URL -O $INSTALL_PATH" && break
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
}

if {{ .ExistsCheck }}; then
  $sh_c "mkdir -p $INSTALL_DIR"

  if [ "$PREFER_DOWNLOAD" = "true" ]; then
    download || inject
  else
    inject || download
  fi

  if [ "$CHMOD_PATH" = "true" ]; then
    $sh_c "chmod +x $INSTALL_PATH"
  fi
fi

# send parent done stream
echo "done"