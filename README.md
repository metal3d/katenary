<div style="text-align:center; margin: auto" align="center">
    <img src="./misc/logo.png" alt="Katenary Logo" style="max-width: 90%" align="center"/>
</div>

Katenary is a tool to help to transform `docker-compose` files to a working Helm Chart for Kubernetes.

> **Important Note:** Katenary is a tool to help to build Helm Chart from a docker-compose file, but docker-compose doesn't propose as many features as what can do Kubernetes. So, we strongly recommend to use Katenary as a "bootstrap" tool and then to manually enhance the generated helm chart.

This project is partially made at [Smile](https://www.smile.eu) 

<div style="text-align:center" align="center">
<a href="https://www.smile.eu"><img src="./misc/Logo_Smile.png" alt="Smile Logo" width="250" /></a>
</div>

# Install

You can download the binaries from the [Release](https://github.com/metal3d/katenary/releases) section. Copy the binary and rename it to `katenary`. Place the binary inside your `PATH`. You should now be able to call the `katenary` command.

You can of course get the binary with `go install -u github.com/metal3d/katenary/cmd/katenary/...` but the `main` branch is continuously updated. It's preferable to use releases.

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

It will use the default PREFIX (`~/.local/`) to install the binary in the `bin` subdirectory. You can force the PREFIX value at install time, but maybe you need to use "sudo":

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

We strongly recommand to add the "completion" call to you SHELL using the common bashrc, or whatever the profile file you use.

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
Katenary aims to be a tool to convert docker-compose files to Helm Charts. 
It will create deployments, services, volumes, secrets, and ingress resources.
But it will also create initContainers based on depend_on, healthcheck, and other features.
It's not magical, sometimes you'll need to fix the generated charts.
The general way to use it is to call one of these commands:

    katenary convert
    katenary convert -c docker-compose.yml
    katenary convert -c docker-compose.yml -o ./charts

In case of, check the help of each command using:
    katenary <command> --help
or
    "katenary help <command>"

Usage:
  katenary [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  convert     Convert docker-compose to helm chart
  help        Help about any command
  show-labels Show labels of a resource
  upgrade     Upgrade katenary to the latest version if available
  version     Display version

Flags:
  -h, --help   help for katenary

Use "katenary [command] --help" for more information about a command.
```

Katenary will try to find a `docker-compose.yaml` or `docker-compose.yml` file inside the current directory. It will check *the existence of the `chart` directory to create a new Helm Chart inside a named subdirectory. Katenary will ask you if you want to delete it before recreating.

It creates a subdirectory inside `chart` that is named with the `appname` option (default is `MyApp`)

> To respect the ability to install the same application in the same namespace, Katenary will create "variable" names like `{{ .Release.Name }}-servicename`. So, you will need to use some labels inside your docker-compose file to help katenary to build a correct helm chart.

What can be interpreted by Katenary:

- Services with "image" section (cannot work with "build" section)
- **Named Volumes** are transformed to persistent volume claims - note that local volume will break the transformation to Helm Chart because there is (for now) no way to make it working (see below for resolution)
- if `ports` and/or `expose` section, katenary will create Services and bind the port to the corresponding container port
- `depends_on` will add init containers to wait for the depending on service (using the first port)
- `env_file` list will create a configMap object per environemnt file (⚠ to-do: the "to-service" label doesn't work with configMap for now)
- some labels can help to bind values, for example:
    - `katenary.io/ingress: 80` will expose the port 80 in an ingress
    - `katenary.io/mapenv: |`: allow mapping environment to something else than the given value in the compose file 

Exemple of a possible `docker-compose.yaml` file:

```yaml
version: "3"
services:
    webapp:
        image: php:7-apache
        environment:
            # note that "database" is a service name
            DB_HOST: database
        expose:
            - 80
        depends_on:
            # this will create a init container waiting for 3306 port
            # because it's the "exposed" port
            - database
        labels:
            # expose the port 80 as an ingress
            katenary.io/ingress: 80
            # make adaptations, DB_HOST environment is actually the service name
            # to hit (note the yaml style, start with "|")
            katenary.io/mapenv: |
              DB_HOST: {{ .Release.Name }}-database
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
            katenary.io/ports: 3306
            # these variables are secrets
            katenary.io/secret-vars: MARIADB_ROOT_PASSWORD, MARIADB_PASSWORD
```

# Labels

These labels could be found by `katenary show-labels`, and can be placed as "labels" inside your docker-compose file:

```
# Labels
katenary.io/ignore               : ignore the container, it will not yied any object in the helm chart (bool)
katenary.io/secret-vars          : secret variables to push on a secret file (coma separated)
katenary.io/secret-envfiles      : set the given file names as a secret instead of configmap (coma separated)
katenary.io/mapenv               : map environment variable to a template string (yaml style, object)
katenary.io/ports                : set the ports to expose as a service (coma separated)
katenary.io/ingress              : set the port to expose in an ingress (coma separated)
katenary.io/configmap-volumes    : specifies that the volumes points on a configmap (coma separated)
katenary.io/same-pod             : specifies that the pod should be deployed in the same pod than the
                                   given service name (string)
katenary.io/volume-from          : specifies that the volumes to be mounted from the given service (yaml style)
katenary.io/empty-dirs           : specifies that the given volume names should be "emptyDir" instead of
                                   persistentVolumeClaim (coma separated)
katenary.io/crontabs             : specifies a cronjobs to create (yaml style, array) - this will create a
                                   cronjob, a service account, a role and a rolebinding to start the command with "kubectl"
                                   The form is the following:
                                   - command: the command to run
                                     schedule: the schedule to run the command (e.g. "@daily" or "*/1 * * * *")
                                     image: the image to use for the command (default to "bitnami/kubectl")
                                     allPods: true if you want to run the command on all pods (default to false)
katenary.io/healthcheck          : specifies that the container should be monitored by a healthcheck,
                                   **it overrides the docker-compose healthcheck**. 
                                   You can use these form of label values:
                                     -> http://[ignored][:port][/path] to specify an http healthcheck
                                     -> tcp://[ignored]:port to specify a tcp healthcheck
                                     -> other string is condidered as a "command" healthcheck
```

# What a name...

Katenary is the stylized name of the project that comes from the "catenary" word.

A catenary is a curve formed by a wire, rope, or chain hanging freely from two points that are not in the same vertical line. For example, the anchor chain between a boat and the anchor.

This "curved link" represents what we try to do, the project is a "streched link from docker-compose to helm chart".


