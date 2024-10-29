# Basic Usage

Basically, you can use `katenary` to transpose a docker-compose file (or any compose file compatible with
`podman-compose` and `docker-compose`) to a configurable Helm Chart. This resulting helm chart can be installed with
`helm` command to your Kubernetes cluster.

!!! Warning "YAML in multiline label"

    Compose only accept text label. So, to put a complete YAML content in the target label, you need to use a pipe char (`|` or `|-`)
    and to **indent** your content.

    For example :

    ```yaml
      labels:
        # your labels
        foo: bar
        # katenary labels with multiline
        katenary.v3/ingress: |-
          hostname: my.website.tld
          port: 80
        katenary.v3/ports: |-
          - 1234
    ```

Katenary transforms compose services this way:

- Takes the service and create a "Deployment" file
- if a port is declared, Katenary creates a service (ClusterIP)
- if a port is exposed, Katenary creates a service (NodePort)
- environment variables will be stored inside a configMap
- image, tags, and ingresses configuration are also stored in `values.yaml` file
- if named volumes are declared, Katenary create PersistentVolumeClaims - not enabled in values file
- `depends_on` needs that the pointed service declared a port. If not, you can use labels to inform Katenary

For any other specific configuration, like binding local files as configMap, bind variables, add values with documentation, etc. You'll need to use labels.

Katenary can also configure containers grouping in pods, declare dependencies, ignore some services, force variables as
secrets, mount files as `configMap`, and many others things. To adapt the helm chart generation, you will need to use
some specific labels.

For more complete label usage, see [the labels page](labels.md).

!!! Info "Overriding file"

    It could be sometimes more convinient to separate the
    configuration related to Katenary inside a secondary file.

    Instead of adding labels inside the `compose.yaml` file,
    you can create a file named `compose.katenary.yaml` and
    declare your labels inside. Katenary will detect it by
    default.

    **No need to precise the file in the command line.**

## Make conversion

After having installed `katenary`, the standard usage is to call:

    katenary convert

It will search standard compose files in the current directory and try to create a helm chart in "chart" directory.

!!! Info
    Katenary uses the compose-go library which respects the Docker and Docker-Compose specification. Keep in mind that
    it will find files exactly the same way as `docker-compose` and `podman-compose` do it.

Of course, you can provide others files than the default with (cumulative) `-c` options:

    katenary convert -c file1.yaml -c file2.yaml

## Some common labels to use

Katenary proposes a lot of labels to configure the helm chart generation, but some are very important.

!!! Info
    For more complete label usage, see [the labels page](labels.md).

### Work with Depends On?

Kubernetes does not provide service or pod starting detection from others pods. But katenary will create init containers
to make you able to wait for a service to respond. But you'll probably need to adapt a bit the compose file.

See this compose file:

```yaml
version: "3"

services:
  webapp:
    image: php:8-apache
    depends_on:
      - database

  database:
    image: mariadb
    environment:
      MYSQL_ROOT_PASSWORD: foobar
```

In this case, `webapp` needs to know the `database` port because the `depends_on` points on it and Kubernetes has not
(yet) solution to check the database startup. Katenary wants to create a `initContainer` to hit on the related service.
So, instead of exposing the port in the compose definition, let's declare this to katenary with labels:

```yaml
version: "3"

services:
  webapp:
    image: php:8-apache
    depends_on:
      - database

  database:
    image: mariadb
    environment:
      MYSQL_ROOT_PASSWORD: foobar
    labels:
      katenary.v3/ports: |-
        - 3306
```

### Declare ingresses

It's very common to have an Ingress resource on web application to deploy on Kubernetes. It allows exposing the
service to the outside of the cluster (you need to install an ingress controller).

Katenary can create this resource for you. You just need to declare the hostname and the port to bind.

```yaml
services:
  webapp:
    image: ...
    ports: 8080:5050
    labels:
      katenary.v3/ingress: |-
        # the target port is 5050 wich is the "service" port
        port: 5050
        hostname: myapp.example.com
```

Note that the port to bind is the one used by the container, not the used locally. This is because Katenary create a
service to bind the container itself.

### Map environment to helm values

A lot of framework needs to receive service host or IP in an environment variable to configure the connection. For
example, to connect a PHP application to a database.

With a compose file, there is no problem as Docker/Podman allows resolving the name by container name:

```yaml
services:
  webapp:
    image: php:7-apache
    environment:
      DB_HOST: database

  database:
    image: mariadb
```

Katenary prefixes the services with `{{ .Release.Name }}` (to make it possible to install the application several times
in a namespace), so you need to "remap" the environment variable to the right one.

```yaml
services:
  webapp:
    image: php:7-apache
    environment:
      DB_HOST: database
    labels:
      katenary.v3/mapenv: |-
        DB_HOST: "{{ .Release.Name }}-database"

  database:
    image: mariadb
```

This label can be used to map others environment for any others reason. E.g. to change an informational environment
variable.

```yaml
services:
  webapp:
    #...
    environment:
      RUNNING: docker
    labels:
      katenary.v3/mapenv: |-
        RUNNING: kubernetes
```

In the above example, `RUNNING` will be set to `kubernetes` when you'll deploy the application with helm, and it's
`docker` for "Podman" and "Docker" executions.
