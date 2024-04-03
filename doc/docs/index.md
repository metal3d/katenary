<div class="md-center"> <img src="statics/logo.png" /> </div>

# Welcome to Katenary documentation


!!! Edit "Thanks to..."
    **Katenary is built with:**
    <br /><a href="https://go.dev" target="_blank">:fontawesome-brands-golang:{ .go-logo }</a>

    **Documentation is built with:**
    <br />
    <a href="https://www.mkdocs.org/" target="_blank">MkDocs</a> using <a
    href="https://squidfunk.github.io/mkdocs-material/" target="_blank">Material for MkDocs</a> theme template.

    > Special thanks to all contributors, testors, and of course packages and tools authors.

Katenary is a tool made to help you to transform "compose" files (`docker-compose.yml`, `podman-compose.yml`...) to
complete and production ready [Helm Chart](https://helm.sh).

You'll be able to deploy your project in [:material-kubernetes: Kubernetes](https://kubernetes.io) in a few seconds 
(of course, more if you need to tweak with labels).

It uses your current file and optionnaly labels to configure the result.

It's an opensource project, under MIT licence, partially developped at [Smile](https://www.smile.eu). The project source 
code is hosted on the [:fontawesome-brands-github: Katenary GitHub Repository](https://github.com/metal3d/katenary).


## Install Katenary

Katenary is developped using the :fontawesome-brands-golang:{ .gopher } [Go](https://go.dev) language. 
The binary is statically linked, so you can simply download it from the [release
page](https://github.com/metal3d/katenary/releases) of the project in GutHub.

You need to select the right binary for your operating system and architecture, and copy the binary in a directory 
that is in your `PATH`.

If you are a Linux user, you can use the "one line installation command" which will download the binary in your 
`$HOME/.local/bin` directory if it exists.

```bash
sh <(curl -sSL https://raw.githubusercontent.com/metal3d/katenary/master/install.sh)
```

!!! Info "Upgrading is integrated to the `katenary` command"
    Katenary propose a `upgrade` subcommand to update the current binary to the latest stable release.

    Of course, you need to install Katenary once :smile:


!!! Note "You prefer to compile it, no need to install Go"
    You can also build and install it yourself, the provided Makefile has got a `build` command that uses `podman` or 
    `docker` to build the binary. 

    So, you don't need to install Go compiler :+1:.

    But, note that the "master" branch is not the "stable" version. It's preferable to switch to a tag, or to use the
    releases.

```bash
git clone https://github.com/metal3d/katenary.git
cd katenary
make build
make install
```

`make install` copies `./katenary` binary to your user binary path (`~/.local/bin`) 

You can install it in other directory by changing the `PREFIX` variable. E.g.:

```bash
make build
sudo make install PREFIX=/usr/local
```

Check if everything is OK using `katenary version` and / or `katenary help`

## Install completion

Katenary uses the very nice project named `cobra` to manage flags, argument and auto-completion.

You can activate it with:
```bash
# replace "bash" by "zsh" if needed
source <(katenary completion bash)
```

Add this line in you `~/.profile` or `~/.bashrc` file to have completion at startup.


