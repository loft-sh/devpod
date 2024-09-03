#!/bin/sh
set -e

on_exit () {
	[ $? -eq 0 ] && exit
	echo 'ERROR: Feature "GitHub CLI" (ghcr.io/devcontainers/features/github-cli) failed to install! Look at the documentation at ${documentation} for help troubleshooting this error.'
}

trap on_exit EXIT

set -a
. ../devcontainer-features.builtin.env
. ./devcontainer-features.env
set +a

echo ===========================================================================

echo 'Feature       : GitHub CLI'
echo 'Description   : Installs the GitHub CLI. Auto-detects latest version and installs needed dependencies.'
echo 'Id            : ghcr.io/devcontainers/features/github-cli'
echo 'Version       : 1.0.13'
echo 'Documentation : https://github.com/devcontainers/features/tree/main/src/github-cli'
echo 'Options       :'
echo '    INSTALLDIRECTLYFROMGITHUBRELEASE="true"
    VERSION="latest"'
echo 'Environment   :'
printenv
echo ===========================================================================

chmod +x ./install.sh
./install.sh
