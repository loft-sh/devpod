#!/bin/sh
set -e

if [ "$SHELL" != "${SHELL%"/zsh"*}" ]; then
  setopt SH_WORD_SPLIT
fi

INSTALL_DIR="{{ .InstallDir }}"
INSTALL_FILENAME="{{ .InstallFilename }}"

INSTALL_PATH="$INSTALL_DIR/$INSTALL_FILENAME"
PREFER_DOWNLOAD="{{ .PreferDownload }}"
CHMOD_PATH="{{ .ChmodPath }}"

# start marker
echo "ping"

# we don't want the script to do anything without us
IFS='$\n' read -r DEVPOD_PING
if [ "$DEVPOD_PING" != "pong" ]; then
  >&2 echo "Received wrong answer for ping request $DEVPOD_PING"
  exit 1
fi

command_exists() {
  command -v "$@" >/dev/null 2>&1
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

inject() {
  echo "ARM-$(is_arm && echo -n 'true' || echo -n 'false')"
  $sh_c "cat > $INSTALL_PATH"

  if [ "$CHMOD_PATH" = "true" ]; then
    $sh_c "chmod +x $INSTALL_PATH"
  fi

  echo "done"
  exit 0
}

download() {
  DOWNLOAD_URL="{{ .DownloadAmd }}"
  if is_arm; then
    DOWNLOAD_URL="{{ .DownloadArm }}"
  fi

  while :; do
    cmd_status=""
    if command_exists curl; then
        $sh_c "curl -fsSL $DOWNLOAD_URL -o $INSTALL_PATH" && break
        cmd_status=$?
    elif command_exists wget; then
        $sh_c "wget -q $DOWNLOAD_URL -O $INSTALL_PATH" && break
        cmd_status=$?
    else
        echo "error: no download tool found, please install curl or wget"
        exit 127
    fi
    >&2 echo "error: failed to download devpod"
    >&2 echo "       command returned: ${cmd_status}"
    >&2 echo "Trying again in 10 seconds..."
    sleep 10
  done
}

if {{ .ExistsCheck }}; then
  user="$(id -un || true)"
  sh_c='sh -c'

  # Try to create the install dir, if we fail, we search for sudo
  # else let's continue without sudo, we don't need it.
  if (! mkdir -p $INSTALL_DIR 2>/dev/null || ! touch $INSTALL_PATH 2>/dev/null || ! chmod +x $INSTALL_PATH 2>/dev/null || ! rm -f $INSTALL_PATH 2>/dev/null); then
    if command_exists sudo; then
      # check if sudo requires a password
      if ! sudo -nl >/dev/null 2>&1; then
        >&2 echo Error: sudo requires a password and no password is available. Please ensure your user account is configured with NOPASSWD.
        exit 1
      fi
      sh_c='sudo -E sh -c'
    elif command_exists su; then
      sh_c='su -c'
    else
      >&2 echo Error: this installer needs the ability to run commands as root.
      >&2 echo We are unable to find either "sudo" or "su" available to make this happen.
      exit 1
    fi

    # Now that we're sudo, try again
    $sh_c "mkdir -p $INSTALL_DIR"
  fi

  $sh_c "rm -f $INSTALL_PATH 2>/dev/null || true"
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

# set download url
export DEVPOD_AGENT_URL={{ .DownloadBase }}

# Execute command
{{ .Command }}
