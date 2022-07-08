# Using labels

Katenary proposes labels to specify adaptation to provide to the Helm Chart. All labels are declared in the help message using:

```text
$ katenary show-labels

# Labels
katenary.io/ignore               : ignore the container, it will not yied any object in the helm chart (bool)
katenary.io/secret-vars          : secret variables to push on a secret file (coma separated)
katenary.io/secret-envfiles      : set the given file names as a secret instead of configmap (coma separated)
katenary.io/mapenv               : map environment variable to a template string (yaml style, object)
katenary.io/ports                : set the ports to assign on the container in pod + expose as a service (coma separated)
katenary.io/container-ports      : set the ports to assign on the contaienr in pod but avoid service (coma separated)
katenary.io/ingress              : set the port to expose in an ingress (coma separated)
katenary.io/configmap-volumes    : specifies that the volumes points on a configmap (coma separated)
katenary.io/same-pod             : specifies that the pod should be deployed in the same pod than the
                                   given service name (string)
katenary.io/volume-from          : specifies that the volumes to be mounted from the given service (yaml style)
katenary.io/empty-dirs           : specifies that the given volume names should be "emptyDir" instead of
                                   persistentVolumeClaim (coma separated)
katenary.io/dependency           : specifies that the given service is actually a Helm Dependency (yaml style)
                                   The form is the following:
                                   - name: name of the dependency
                                     version: version of the dependency
                                     repository: repository of the dependency
                                     alias: alias of the dependency (optional)
                                     config: config of the dependency (map, optional)
                                       environment: map for environment
                                       serviceName: the service name as defined in the chart that replace the current service name (default to the compose service name)
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

## healthcheck

HealthCheck label defines how to make LivenessProbe on Kubernetes.

!!! Warning
    This overrides the compose file healthcheck

!!! Info
    The hostname is set to "localhost" by convention, but Katenary will ignore the hostname in tcp and http tests because it will create a LivenessProbe.

Some example of usage:

```yaml
services:
  mariadb:
    image: mariadb
    labels:
      katenary.io/healthcheck: tcp://localhost:3306

  webapp:
    image: nginx
    labels:
      katenary.io/healthcheck: http://localhost:80

  example:
    image: yourimage
    labels:
      katenary.io/healthcheck: "test -f /opt/installed"
```

## crontabs

Crontabs label proposes to create a complete CronTab object with needed RBAC to make it possible to run command inside the pod(s) with `kubectl`. Katenary will make the job for you. You only need to provide the command(s) to call.

It's a YAML array in multiline label. 

```yaml
services:
  mariadb:
    image: mariadb
    labels:
      katenary.io/crontabs: |
        - command: mysqldump -B myapp -uroot -p$${MYSQL_ROOT_PASSWORD} > dump.sql
          schedule: "@every 1h"
```
The object is:
```
command:  Command to run
schedule: the cron form schedule string
allPods:  boolean (default false) to activate the cront on each pod
image:    image name to use (default is bitnami/kubectl) 
          with corresponding tag to your kubernetes version
```

## empty-dirs

You sometime don't need to create a PersistentVolumeClaim. For example when a volume in your compose file is actually made to share the data between 2 or more containers.

In this case, an "emptyDir" volume is appreciated.

```yaml
services:
  webapp:
    image: nginx
    volumes:
    - websource:/var/www/html
    labels:
      # sources is actually an empty directory on the node
      katenary.io/empty-dirs: websource

  php:
    image: php:7-fpm
    volumes:
    - sources:/var/www/html
    labels:
      # in the same pod than webapp
      katenary.io/same-pod: webapp
      # see the corresponding section, get the volume
      # fro webapp
      katenary.io/volume-from: |
        sources:
        webapp: websource
```

## volume-from

We see this in the [empty-dirs](#empty-dirs) section, this label defines that the corresponding volume should be shared in this pod.

```yaml
services:
  webapp:
    image: nginx
    volumes:
    - datasource:/var/www/html

  app:
    image: php
    volumes:
    - data:/opt/data
    labels:
      katenary.io/volume-from: |
        # data in this container...
        data:
        # ... correspond to "datasource" in "webapp" container
        webapp: datasource
```

This implies that the declared volume in "webapp" will be mounted to "app" pods.

!!! Warning
    This is possible with Kubernetes volumes restrictions. So, it works in these cases:

    - if the volume class is Read Write Many
    - or if you mount the volume in the same pod (so in the same node)
    - and/or the volume is an emptyDir


## same-pod

It's sometimes important and/or necessary to declare that 2 services are in the same pod. For example, using PHP-FPM and NGinx. In this case, you can declare that both services are in the same pod.

You must declare this label only on "supplementary" services and always use the same master service for the entire pod declaration.

```yaml
services:
  web:
    image: nginx

  php:
    image: php:8-fpm
    labels:
      katenary.io/same-pod: web
```

The above example will create a `web` deployment, the PHP container is added in the `web` pod.

## configmap-volumes

This label proposes to declare a file or directory where content is actually static and can be mounted as configMap volume.

It's a comma separated label, you can declare several volumes.

For example, in `static/index.html`:

```html
<html>
<body>Hello</body>
</html>
```

And a compose file (snippet):

```yaml
serivces:
  web:
    image: nginx
    volumes:
    - ./static:/usr/share/nginx/html:z
    labels:
      katenary.io/configmap-volumes: ./statics
```

What will make Katenary:

- create a configmap containing the "index.html" file as data
- declare the volume in the `web` deployment file
- mount the configmap in `/usr/share/nginx/html` directory of the container

## ingress

Declare which port to use to create an ingress. The hostname will be declared in `values.yaml` file.

```yaml
serivces:
  web:
    image: nginx
    ports:
    - 8080:80
    labels:
      katenary.io/ingress: 80
```

!!! Info
    A port **must** be declared, in `ports` section or with `katenary.io/ports` label. This to force the creation of a `Service`.

## ports and container-ports

This changes or set the `Service` port declared in the Helm Chart.

Exposing or declaring ports in a compose file is not mandatory. Others containers may contact the container port because there is no "Pod" notion. And you probably don't wanto to explicitally declare these ports if you don't need them in a Docker/Podman context.

> But Katenary will need to know the ports if you declared `depends_on` or if you need to contact the `Pod` from another one. That's because Kubernetes uses `Service` objects to load balance connexions to the `Pod`.

In this case, you can declare the ports in the corresponding label, these will force the creation of a `Service` listening on the given ports (and making the load balancing to the `Pod` ports):

```yaml
serivces:
  web:
    image: nginx
    labels:
      katenary.io/ports: 80,443
```

This will leave Katenary creating the service to open these ports to others pods.

Another case is that you need to have `containerPort` in pods but **avoid the service declaration**, so you can use this label:

```yaml
services:
  php:
    image: php:8-fpm
    labels:
      katenary.io/container-ports: 9000
```

That will only declare the container port in the pod, but **not in the service**.

!!! Info
    It's very useful when you need to declare ports in conjonction with `same-pod`. Katenary would create a service with all the pods ports inside. The `container-ports` label will make the ports to be ignored in the service creation.

## mapenv

Environment variables are working great for your compose stack, but you sometimes need to change them in Helm. This label allows you to remap the value for Helm.

For example, when you use an environment variable to point on another service.

```yaml
serivces:
  php:
    image: php
    environment:
      DB_HOST: database

  database:
    image: mariadb
    labels:
      katenary.io/ports: 3306
```

The above example will break when you'll start it in Kubernetes because the `database` service will not be named like this, it will be renamed to `{{ .Release.Name }}-database`. So, you can declare the rewrite:

```yaml
services:
  php:
    image: php
    environment:
      DB_HOST: database
    labels:
      katenary.io/mapenv: |
        DB_HOST: "{{ .Release.Name }}"-database
  database:
    image: mariadb
    labels:
      katenary.io/ports: 3306

```

It's also useful when you want to change a variable value to another when you deploy on Kubernetes.

## secret-envfiles

Katenary binds all "environemnt files" to config maps. But some of these files can be bound as sercrets.

In this case, declare the files as is:

```yaml
services:
  app:
    image: #...
    env_file:
      - ./env/whatever
      - ./env/sensitives
    labels:
      katenary.io/secret-envfiles: ./env/sensitives
```

## secret-vars

If you have some environemnt variables to declare as secret, you can list them in the `secret-vars` label.

```yaml
services:
  database:
    image: mariadb
    environemnt:
      MYSQL_PASSWORD: foobar
      MYSQL_ROOT_PASSWORD: longpasswordhere
      MYSQL_USER: john
      MYSQL_DATABASE: appdb
    labels:
      katenary.io/secret-vars: MYSQL_ROOT_PASSWORD,MYSQL_PASSWORD
```

## ignore

Simply ignore the service to not be exported in the Helm Chart.

```yaml
serivces:

  # this service is able to answer HTTP
  # on port 5000
  webapp:
  image: myapp
  labels:
    # declare the port
    katenary.io/ports: 5000
    # the ingress controller is a web proxy, so...
    katenary.io/ingress: 5000


  # with local Docker, I want to access my webapp
  # with "myapp.locahost" so I use a nice proxy on
  # port 80
  proxy:
    image: quay.io/pathwae/proxy
    ports:
    - 80:80
    environemnt:
      CONFIG: |
        myapp.localhost: webapp:5000
    labels:
      # I don't need it in Helm, it's only
      # for local test!
      katenary.io/ignore: true
```

## dependency

Replace the service by a [Helm Chart Dependency](https://helm.sh/docs/helm/helm_dependency/).

!!! Warning "Don't forget to update"
    You need to launch `helm dep update` to download the chart before to be able to test or deploy your generated helm chart

The dependency respects the `Chart.yaml` file form + an "environemnt" block that will be set inside the `values.yaml` file.

```yaml
services:
  myapp:
    image: php:8-apache
    depends_on:
    - mariadb

  mariadb:
    # this for docker-compose
    image: mariadb
    labels:
      # only for the depends_on directive
      katenary.io/ports: 3306
      # replace this by a "helm dependency"
      katenary.io/dependency: |
        name: mariadb-galera
        repository: https://charts.bitnami.com/bitnami
        version: 10.6.x
        # alias: database
        #
        # Configuration:
        # => helm show values bitnami/mariadb-galera
        config:
          # serviceName: {{ .Release.Name }}-mariadb-galera
          environment:
            rootUser:
              password: TheRootPassword
            db:
              user: user1
              password: theuserpassword
              name: myapp
```

When katenary parses the docker compose file, it will replace the `myapp` `depends_on` list to match the helm dependency name. This because the majority of helm chart uses the helm chart name as service name. Of course, as usual, the `{{ .Release.Name }}` is set as prefix.

For a few helm charts, this cannot be applied because the service name is not defined with this rule. So you can override the service name with `serviceName`.

