# Basic Usage

Basically, you can use `katenary` to transpose a docker-compose file (or any compose file compatible with `podman-compose` and `docker-compose`) to a configurable Helm Chart. This resulting helm chart can be installed with `helm` command to your Kubernetes cluster.

Katenary transforms compose services this way:

- Takes the service and create a "Deployment" file
- if a port is declared, katenary creates a service (ClusterIP)
- it a port is exposed, katenary creates a service (NodePort)
- environment variables will be stored in `values.yaml` file
- image, tags, and ingresses configuration are also stored in `values.yaml` file
- if named volumes are declared, katenary create PersistentVolumeClaims - not enabled in values file (a `emptyDir` is used by default)
- any other volume (local mount points) are ignored
- `depends_on` needs that the pointed service declared a port. If not, you can use labels to inform katenary

Katenary can also configure containers grouping in pods, declare dependencies, ignore some services, force variables as secrets, mount files as `configMap`, and many others things. To adapt the helm chart generation, you will need to use some specific labels.

## Try to convert

After having installed `katenary`, the standard usage is to call:

```bash
katenary convert
```

It will search standard compose files in the current directory and try to create a helm chart in "chart" directory.

Katenary respects the Docker rules for overrides files, and you can of course force others files:
```bash
katenary convert -c file1.yaml -c file2.yaml
```


## Work with Depends On?

Kubernetes does not propose service or pod starting detection from others pods. But katenary will create init containers to make you able to wait for a service to respond. But you'll probably need to adapt a bit the compose file.

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

In this case, `webapp` needs to know the `database` port because the `depends_on` points on it and Kubernetes has not (yet) solution to check the database startup. Katenary wants to create a `initContainer` to hit on the related service. So, instead of exposing the port in the compose definition, let's declare this to katenary with labels:


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
            katenary.io/ports: 3306
```

