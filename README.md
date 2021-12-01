Katenary is a tool to help transforming `docker-compose` files to a working Helm Chart for Kubernetes.


# Install

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


# Usage

```bash
Usage of katenary:
  -appname string
    	sive the helm chart app name (default "MyApp")
  -appversion string
    	set the chart appVersion (default "0.0.1")
  -chart-dir string
    	set the chart directory (default "chart")
  -compose string
    	set the compose file to parse (default "docker-compose.yaml")
  -force
    	force the removal of the chart-dir
  -version
    	Show version and exit
```

Katenary will try to find a `docker-compose.yaml` file inside the current directory. It will check *the existence of the `chart` directory to create a new Helm Chart inside a named subdirectory. Katenary will ask you if you want to delete it before recreating.

It creates a subdirectory inside `chart` that is named with the `appname` option (default is `MyApp`)

> To respect the ability to install the same application in the same namespace, Katenary will create "variable" names like `{{ .Release.Name }}-servicename`. So, you will need to use some labels inside your docker-compose file to help katenary to build a correct helm chart.

What can be interpreted by Katenary:

- Services with "image" section (cannot work with "build" section)
- **Named Volumes** are transformed to persistent volume claims - note that local volume will break the transformation to Helm Chart because there is (for now) no way to make it working (see below for resolution)
- if `ports` and/or `expose` section, katenary will create Services and bind the port to the corresponding container port
- `depends_on` will add init containers to wait for the depending service (using the first port)
- `env_file` list will create a configMap object per environemnt file (⚠ todo: the "to-service" label doesn't work with configMap for now)
- some labels can help to bind values:
    - `katenary.io/expose-ingress: true` will expose the first port or expose to an ingress
    - `katenary.io/to-service: VARNAME` will convert the value to a variable `{{ .Release.Name }}-VARNAME` - it's usefull when you want to pass the name of a service as a variable (think about the service name for mysql to pass to a container that wants to connect to this)

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
            # explain to katenary that "DB_HOST" value is variable (using release name)
            katenary.io/to-servie: DB_HOST
    database:
        image: mariabd:10
        env_file:
            # this will create a configMap
            - my_env.env
        environment:
            MARIADB_ROOT_PASSWORD: foobar
        expose:
            # required to make katenary able to create the init container
            - 3306
```

# Labels

- `katenary.io/to-service` binds the given (coma separated) variables names  to {{ .Release.Name }}-value
- `katenary.io/expose-ingress`: create an ingress and bind it to the service
