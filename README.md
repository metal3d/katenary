<div style="text-align:center; margin: auto 0 4em 0" align="center">
<img src="./doc/docs/statics/logo-vertical.svg" alt="Katenary Logo" style="max-width: 90%" align="center"/>
</div>

[![Documentation Status](https://readthedocs.org/projects/katenary/badge/?version=latest)](https://katenary.readthedocs.io/en/latest/?badge=latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/metal3d/katenary)](https://goreportcard.com/report/github.com/metal3d/katenary)
[![GitHub release](https://img.shields.io/github/v/release/metal3d/katenary)](https://github.com/metal3d/katenary/releases)

🚀 Unleash Productivity with Katenary! 🚀

Tired of manual conversions? Katenary harnesses the labels from your "compose" file to craft complete Helm Charts 
effortlessly, saving you time and energy.

🛠️ Simple autmated CLI: Katenary handles the grunt work, generating everything needed for seamless service binding 
and Helm Chart creation.

💡 Effortless Efficiency: You only need to add labels when it's necessary to precise things. Then call `katenary convert` and let the magic happen.

# What ?

Katenary is a tool to help to transform `docker-compose` files to a working Helm Chart for Kubernetes.

> **Important Note:** Katenary is a tool to help to build Helm Chart from a docker-compose file, but docker-compose
> doesn't propose as many features as what can do Kubernetes. So, we strongly recommend to use Katenary as a "bootstrap"
> tool and then to manually enhance the generated helm chart.


Today, it's partially developped in collaboration with [Klee Group](https://www.kleegroup.com). Note that Katenary is
and **will stay an opensource and free (as freedom) project**. We are convinced that the best way to make it better is to
share it with the community.

The main developer is [Patrice FERLET](https://github.com/metal3d).

# Install

You can download the binaries from the [Release](https://github.com/metal3d/katenary/releases) section. Copy the binary
and rename it to `katenary`. Place the binary inside your `PATH`. You should now be able to call the `katenary` command.

You can of course get the binary with `go install -u github.com/metal3d/katenary/cmd/katenary/...` but the `main` branch
is continuously updated. It's preferable to use releases.

You can use this commands on Linux:

```bash
sh <(curl -sSL https://raw.githubusercontent.com/metal3d/katenary/master/install.sh)
```

# Else... Build yourself

If you've got `podman` or `docker`, you can build `katenary` by using:

```bash
make build
```

You can then install it with:
```bash
make install
```

It will use the default PREFIX (`~/.local/`) to install the binary in the `bin` subdirectory. You can force the PREFIX
value at install time, but maybe you need to use "sudo":

```bash
sudo make install PREFIX=/usr/local
```

If that goes wrong, you can use your local Go compiler:

```bash
make build GO=local

# To force OS or architecture
make build GO=local GOOS=linux GOARCH=arm64
```

Then place the `katenary` binary file inside your PATH.


# Tips

We strongly recommand to add the "completion" call to you SHELL using the common bashrc, or whatever the profile file
you use.

E.g.:

```bash
# bash in ~/.bashrc file
source <(katenary completion bash)
# if the documentation breaks a bit your completion:
source <(katenary completion bash --no-description)

# zsh in ~/.zshrc
source <(katenary completion zsh)

# fish in ~/.config/fish/config.fish
  katenary completion fish | source

# powershell (as we don't provide any support on Windows yet, please avoid this...)
  ```

# Usage

```
Katenary is a tool to convert compose files to Helm Charts.

Each [command] and subcommand has got an "help" and "--help" flag to show more information.

Usage:
katenary [command]

Examples:
katenary convert -c docker-compose.yml -o ./charts

Available Commands:
completion        Generates completion scripts
convert           Converts a docker-compose file to a Helm Chart
hash-composefiles Print the hash of the composefiles
help              Help about any command
help-labels       Print the labels help for all or a specific label
version           Print the version number of Katenary

Flags:
-h, --help      help for katenary
-v, --version   version for katenary

Use "katenary [command] --help" for more information about a command.
```

  Katenary will try to find a `docker-compose.yaml` or `docker-compose.yml` file inside the current directory. It will
  check *the existence of the `chart` directory to create a new Helm Chart inside a named subdirectory. Katenary will ask
  you if you want to delete it before recreating.

It creates a subdirectory inside `chart` that is named with the `appname` option (default is `MyApp`)

  > To respect the ability to install the same application in the same namespace, Katenary will create "variable" names
  > like `{{ .Release.Name }}-servicename`. So, you will need to use some labels inside your docker-compose file to help
  > katenary to build a correct helm chart.

  What can be interpreted by Katenary:

  - Services with "image" section (cannot work with "build" section)
  - **Named Volumes** are transformed to persistent volume claims - note that local volume will break the transformation
to Helm Chart because there is (for now) no way to make it working (see below for resolution)
  - if `ports` and/or `expose` section, katenary will create Services and bind the port to the corresponding container port
- `depends_on` will add init containers to wait for the depending on service (using the first port)
  - `env_file` list will create a configMap object per environemnt file (⚠ to-do: the "to-service" label doesn't work with
      configMap for now)
  - some labels can help to bind values, see examples below

  Exemple of a possible `docker-compose.yaml` file:

```yaml
services:
  webapp:
    image: php:7-apache
    environment:
      # note that "database" is a "compose" service name
      # so we need to adapt it with the map-env label
      DB_HOST: database
    expose:
    - 80
    depends_on:
      # this will create a init container waiting for 3306 port
      # because it's the "exposed" port
      - database
    labels:
      # expose the port 80 as an ingress
      katenary.v3/ingress: |-
        hostname: myapp.example.com
        port: 80
      katenary.v3/mapenv: |-
        # make adaptations, DB_HOST environment is actually the service name
        DB_HOST: '{{ .Release.Name }}-database'

    database:
      image: mariadb:10
      env_file:
        # this will create a configMap
        - my_env.env
      environment:
        MARIADB_USER: foo
        MARIADB_ROOT_PASSWORD: foobar
        MARIADB_PASSWORD: bar
      labels:
        # no need to declare this port in docker-compose
        # but katenary will need it
        katenary.v3/ports: |-
          - 3306
        # these variables are secrets
        katenary.v3/secrets: |-
          - MARIADB_ROOT_PASSWORD
          - MARIADB_PASSWORD
```

# Labels

These labels could be found by `katenary help-labels`, and can be placed as "labels" inside your docker-compose file:

```
To get more information about a label, use `katenary help-label <name_without_prefix>
e.g. katenary help-label dependencies

katenary.v3/configmap-files:	list of strings		Add files to the configmap.
katenary.v3/cronjob:		object			Create a cronjob from the service.
katenary.v3/dependencies:	list of objects		Add Helm dependencies to the service.
katenary.v3/description:	string			Description of the service
katenary.v3/env-from:		list of strings		Add environment variables from antoher service.
katenary.v3/health-check:	object			Health check to be added to the deployment.
katenary.v3/ignore:		bool			Ignore the service
katenary.v3/ingress:		object			Ingress rules to be added to the service.
katenary.v3/main-app:		bool			Mark the service as the main app.
katenary.v3/map-env:		object			Map env vars from the service to the deployment.
katenary.v3/ports:		list of uint32		Ports to be added to the service.
katenary.v3/same-pod:		string			Move the same-pod deployment to the target deployment.
katenary.v3/secrets:		list of string		Env vars to be set as secrets.
katenary.v3/values:		list of string or map	Environment variables to be added to the values.yaml
```

# What a name...

Katenary is the stylized name of the project that comes from the "catenary" word.

A catenary is a curve formed by a wire, rope, or chain hanging freely from two points that are not in the same vertical
line. For example, the anchor chain between a boat and the anchor.

This "curved link" represents what we try to do, the project is a "streched link from docker-compose to helm chart".
