# Home

Welcome to the documentation of Katenary.

## What is Katenary?

Katenary is a project that aims to help you to transform "compose" files (`docker-compose.yml`, `podman-compose.yml`...) to a complete and production ready [Helm Chart](https://helm.sh).

It uses your current file and optionnaly labels to configure the result.

It's an opensource project, under MIT licence, partially developped at [Smile](https://smile.eu). The project source code is hosted on the [Katenary GitHub Repository](https://github.com/metal3d/katenary).

## Install Katenary

Katenary is developped in [Go](https://go.dev). The binary is statically linked, so you can only download it from the [release page](https://github.com/metal3d/katenary/releases) of the project in GutHub.

You need to select the right binary for your operating system and architecture, and copy the binary in a directory that is in your `PATH`.

If you are a Linux user, you can use the "one line installation command" which will download the binary in you `$HOME/.local/bin` directory if it exists.

```bash
sh <(curl -sSL https://raw.githubusercontent.com/metal3d/katenary/master/install.sh)
```

You can also build and install it yourself, the giver Makefile provides a `build` command that uses `podman` or `docker` to build the binary. You don't need to install Go compiler so.

```bash
git clone https://github.com/metal3d/katenary.git
cd katenary
make build
```

Then, copy `./katenary` binary to you `PATH` (`~/.local/bin` or `/usr/local/bin` with `sudo`) and type `katenary version` and / or `katenary help`

## Install completion

Katenary uses the very nice project named `cobra` to manage flags, argument and auto-completion.

You can activate it with:
```bash
# replace "bash" by "zsh" if needed
source <(katenary completion bash)
```

Add this line in you `~/.profile` or `~/.bashrc` file to have completion at startup.


