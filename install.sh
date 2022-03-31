#!/bin/sh

# Install the latest version of the Katenary detecting the right OS and architecture.
# Can be launched with the following command:
# sh <(curl -sSL https://raw.githubusercontent.com/metal3d/katenary/master/install.sh)

set -e

# Detect the OS and architecture
OS=$(uname)
ARCH=$(uname -m)

# Detect where to install the binary, local path is the prefered method
INSTALL_TYPE=$(echo $PATH | grep "$HOME/.local/bin" 2>&1 >/dev/null && echo "local" || echo "global")

# Where to download the binary
BASE="https://github.com/metal3d/katenary/releases/latest/download/"


if [ $ARCH = "x86_64" ]; then
    ARCH="amd64"
fi

BIN_URL="$BASE/katenary-$OS-$ARCH"

INSTALL_TYPE="global"
if [ "$INSTALL_TYPE" = "local" ]; then
    echo "Installing to local directory, installing in $HOME/.local/bin"
    BIN_PATH="$HOME/.local/bin"
else
    echo "Installing to global directory, installing in /usr/local/bin - we need to use sudo..."
    answer=""
    while [ "$answer" != "y" ] && [ "$answer" != "n" ]; do
        echo -n "Are you OK? [y/N] "
        read answer
        # lower case answer
        answer=$(echo $answer | tr '[:upper:]' '[:lower:]')
        if [ "$answer" == "n" ] || [ -z "$answer" ]; then
            echo "--> To install locally, please ensure that \$HOME/.local/bin is in your PATH"
            echo "Cancelling installation"
            exit 0
        fi
    done
    BIN_PATH="/usr/local/bin"
fi

echo
echo "Downloading $BIN_URL"
USE_SUDO=$([ "$INSTALL_TYPE" = "local" ] && echo "" || echo "sudo")

T=$(mktemp -u)
$USE_SUDO curl -SL -# $BIN_URL -o $T || (echo "Failed to download katenary" && rm -f $T && exit 1)

$USE_SUDO mv $T $BIN_PATH/katenary
$USE_SUDO chmod +x $BIN_PATH/katenary
echo
echo "Installed to $BIN_PATH/katenary"
echo "Installation complete! Run 'katenary --help' to get started."
