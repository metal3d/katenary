#!/bin/sh

# Install the latest version of the Katenary detecting the right OS and architecture.
# Can be launched with the following command:
# sh <(curl -sSL https://raw.githubusercontent.com/metal3d/katenary/master/install.sh)

set -e

# Detect the OS and architecture
OS=$(uname)
ARCH=$(uname -m)

# Detect the home directory "bin" directory, it is commonly:
# - $HOME/.local/bin
# - $HOME/.bin
# - $HOME/bin
COMON_INSTALL_PATHS="$HOME/.local/bin $HOME/.bin $HOME/bin"

INSTALL_PATH=""
for p in $COMON_INSTALL_PATHS; do
    if [ -d $p ]; then
        INSTALL_PATH=$p
        break
    fi
done

# check if the user has write access to the INSTALL_PATH
if [ -z "$INSTALL_PATH" ]; then
    INSTALL_PATH="/usr/local/bin"
    if [ ! -w $INSTALL_PATH ]; then
        echo "You don't have write access to $INSTALL_PATH"
        echo "Please, run with sudo or install locally"
        exit 1
    fi
fi

# ensure that $INSTALL_PATH is in the PATH
if ! echo $PATH | grep -q $INSTALL_PATH; then
    echo "Sorry, $INSTALL_PATH is not in the PATH"
    echo "Please, add it to your PATH in your shell configuration file"
    echo "then restart your shell and run this script again"
    exit 1
fi

# Where to download the binary
BASE="https://github.com/metal3d/katenary/releases/latest/download/"

# for compatibility with older ARM versions
if [ $ARCH = "x86_64" ]; then
    ARCH="amd64"
fi

BIN_URL="$BASE/katenary-$OS-$ARCH"

echo
echo "Downloading $BIN_URL"

T=$(mktemp -u)
curl -SL -# $BIN_URL -o $T || (echo "Failed to download katenary" && rm -f $T && exit 1)

mv $T $INSTALL_PATH/katenary
chmod +x $INSTALL_PATH/katenary
echo
echo "Installed to $INSTALL_PATH/katenary"
echo "Installation complete! Run 'katenary help' to get started."
