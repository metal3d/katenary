# Labels documentation

Katenary proposes labels to set in `compose.yaml` files (or override files) to configure the Helm Chart generation. Because it is sometimes needed to have structured values, it is necessary to use the Yaml syntax. While compose labels are string, we can use `|` to use Yaml multilines as value.

Katenary will try to Unmarshal these labels.

## Label list and types

<!-- START_LABEL_DOC : do not remove this tag !-->
| Label name                     | Description                                                      | Type                             |
| ------------------------------ | ---------------------------------------------------------------- | -------------------------------- |
| `katenary.v3/configmap-files`  | Add files to the configmap.                                      | `[]string`                       |
| `katenary.v3/cronjob`          | Create a cronjob from the service.                               | `object`                         |
| `katenary.v3/dependencies`     | Add Helm dependencies to the service.                            | `[]object`                       |
| `katenary.v3/description`      | Description of the service                                       | `string`                         |
| `katenary.v3/env-from`         | Add environment variables from antoher service.                  | `[]string`                       |
| `katenary.v3/exchange-volumes` | Add exchange volumes (empty directory on the node) to share data | `[]object`                       |
| `katenary.v3/health-check`     | Health check to be added to the deployment.                      | `object`                         |
| `katenary.v3/ignore`           | Ignore the service                                               | `bool`                           |
| `katenary.v3/ingress`          | Ingress rules to be added to the service.                        | `object`                         |
| `katenary.v3/main-app`         | Mark the service as the main app.                                | `bool`                           |
| `katenary.v3/map-env`          | Map env vars from the service to the deployment.                 | `map[string]string`              |
| `katenary.v3/ports`            | Ports to be added to the service.                                | `[]uint32`                       |
| `katenary.v3/same-pod`         | Move the same-pod deployment to the target deployment.           | `string`                         |
| `katenary.v3/secrets`          | Env vars to be set as secrets.                                   | `[]string`                       |
| `katenary.v3/values`           | Environment variables to be added to the values.yaml             | `[]string or map[string]string`  |
| `katenary.v3/values-from`      | Add values from another service.                                 | `map[string]string`              |

<!-- STOP_LABEL_DOC : do not remove this tag !-->

## Detailed description

<!-- START_DETAILED_DOC : do not remove this tag !-->
### katenary.v3/configmap-files

Add files to the configmap.

**Type**:  `[]string`

It makes a file or directory to be converted to one or more ConfigMaps 
and mounted in the pod. The file or directory is relative to the 
service directory.

If it is a directory, all files inside it are added to the ConfigMap.

If the directory as subdirectories, so one configmap per subpath are created.

!!! Warning
    It is not intended to be used to store an entire project in configmaps.
    It is intended to be used to store configuration files that are not managed 
    by the application, like nginx configuration files. Keep in mind that your
    project sources should be stored in an application image or in a storage.

**Example:**

```yaml
volumes
  - ./conf.d:/etc/nginx/conf.d
labels:
  katenary.v3/configmap-files: |-
    - ./conf.d
```


### katenary.v3/cronjob

Create a cronjob from the service.

**Type**:  `object`

This adds a cronjob to the chart.

The label value is a YAML object with the following attributes:
- command: the command to be executed 
- schedule: the cron schedule (cron format or @every where "every" is a 
  duration like 1h30m, daily, hourly...)
- rbac: false (optionnal), if true, it will create a role, a rolebinding and
  a serviceaccount to make your cronjob able to connect the Kubernetes API

**Example:**

```yaml
labels:
    katenary.v3/cronjob: |-
        command: echo "hello world"
        schedule: "* */1 * * *" # or @hourly for example
```


### katenary.v3/dependencies

Add Helm dependencies to the service.

**Type**:  `[]object`

Set the service to be, actually, a Helm dependency. This means that the 
service will not be exported as template. The dependencies are added to 
the Chart.yaml file and the values are added to the values.yaml file.

It's a list of objects with the following attributes:

- name: the name of the dependency
- repository: the repository of the dependency
- alias: the name of the dependency in values.yaml (optional)
- values: the values to be set in values.yaml (optional)

!!! Info
    Katenary doesn't update the helm depenedencies by default.
    
    Use `--helm-update` (or `-u`) flag to update the dependencies.
    
    example: <code>katenary convert -u</code>

By setting an alias, it is possible to change the name of the dependency 
in values.yaml.

**Example:**

```yaml
labels:
  katenary.v3/dependencies: |-
    - name: mariadb
      repository: oci://registry-1.docker.io/bitnamicharts

      ## optional, it changes the name of the section in values.yaml
      # alias: mydatabase

      ## optional, it adds the values to values.yaml
      values:
        auth:
          database: mydatabasename
          username: myuser
          password: the secret password
```


### katenary.v3/description

Description of the service

**Type**:  `string`

This replaces the default comment in values.yaml file to the given description. 
It is useful to document the service and configuration.

The value can be set with a documentation in multiline format.

**Example:**

```yaml
labels:
  katenary.v3/description: |-
    This is a description of the service.
    It can be multiline.
```


### katenary.v3/env-from

Add environment variables from antoher service.

**Type**:  `[]string`

It adds environment variables from another service to the current service.

**Example:**

```yaml
service1:
  image: nginx:1.19
  environment:
      FOO: bar

service2:
  image: php:7.4-fpm
  labels:
    # get the congigMap from service1 where FOO is 
    # defined inside this service too
    katenary.v3/env-from: |-
        - myservice1
```


### katenary.v3/exchange-volumes

Add exchange volumes (empty directory on the node) to share data

**Type**:  `[]object`

This label allows sharing data between containres. The volume is created in 
the node and mounted in the pod. It is useful to share data between containers
in a "same pod" logic. For example to let PHP-FPM and Nginx share the same direcotory.

This will create:

- an `emptyDir` volume in the deployment
- a `voumeMount` in the pod for **each container**
- a `initContainer` for each definition

Fields:
  - name: the name of the volume (manadatory)
  - mountPath: the path where the volume is mounted in the pod (optional, default is `/opt`)
  - init: a command to run to initialize the volume with data (optional)

!!! Warning
    This is highly experimental. This is mainly useful when using the "same-pod" label.

**Example:**

```yaml
nginx:
  # ...
  labels;
    katenary.v3/exchange-volumes: |-
      - name: php-fpm
        mountPath: /var/www/html
php:
  # ...
  labels:
    katenary.v3/exchange-volumes: |-
      - name: php-fpm
        mountPath: /opt
        init: cp -ra /var/www/html/* /opt
```


### katenary.v3/health-check

Health check to be added to the deployment.

**Type**:  `object`

Health check to be added to the deployment.

**Example:**

```yaml
labels:
  katenary.v3/health-check: |-
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
```


### katenary.v3/ignore

Ignore the service

**Type**:  `bool`

Ingoring a service to not be exported in helm chart.

**Example:**

```yaml
labels:
  katenary.v3/ignore: "true"
```


### katenary.v3/ingress

Ingress rules to be added to the service.

**Type**:  `object`

Declare an ingress rule for the service. The port should be exposed or 
declared with `katenary.v3/ports`.

**Example:**

```yaml
labels:
  katenary.v3/ingress: |-
    port: 80
    hostname: mywebsite.com (optional)
```


### katenary.v3/main-app

Mark the service as the main app.

**Type**:  `bool`

This makes the service to be the main application. Its image tag is 
considered to be the Chart appVersion and to be the defaultvalue in Pod
container image attribute.

!!! Warning
    This label cannot be repeated in others services. If this label is
    set in more than one service as true, Katenary will return an error.

**Example:**

```yaml
ghost:
  image: ghost:1.25.5
  labels:
    # The chart is now named ghost, and the appVersion is 1.25.5.
    # In Deployment, the image attribute is set to ghost:1.25.5 if 
    # you don't change the "tag" attribute in values.yaml
    katenary.v3/main-app: true
```


### katenary.v3/map-env

Map env vars from the service to the deployment.

**Type**:  `map[string]string`

Because you may need to change the variable for Kubernetes, this label
forces the value to another. It is also particullary helpful to use a template 
value instead. For example, you could bind the value to a service name 
with Helm attributes:
`{{ tpl .Release.Name . }}`.

If you use `__APP__` in the value, it will be replaced by the Chart name.

**Example:**

```yaml
env:
  DB_HOST: database
  RUNNING: docker
  OTHER: value
labels:
  katenary.v3/map-env: |-
    RUNNING: kubernetes
    DB_HOST: '{{ include "__APP__.fullname" . }}-database'
```


### katenary.v3/ports

Ports to be added to the service.

**Type**:  `[]uint32`

Only useful for services without exposed port. It is mandatory if the 
service is a dependency of another service.

**Example:**

```yaml
labels:
  katenary.v3/ports: |-
    - 8080
    - 8081
```


### katenary.v3/same-pod

Move the same-pod deployment to the target deployment.

**Type**:  `string`

This will make the service to be included in another service pod. Some services 
must work together in the same pod, like a sidecar or a proxy or nginx + php-fpm.

Note that volume and VolumeMount are copied from the source to the target 
deployment.

**Example:**

```yaml
web:
  image: nginx:1.19

php:
  image: php:7.4-fpm
  labels:
    katenary.v3/same-pod: web
```


### katenary.v3/secrets

Env vars to be set as secrets.

**Type**:  `[]string`

This label allows setting the environment variables as secrets. The variable 
is removed from the environment and added to a secret object.

The variable can be set to the `katenary.v3/values` too,
so the secret value can be configured in values.yaml

**Example:**

```yaml
env:
  PASSWORD: a very secret password
  NOT_A_SECRET: a public value
labels:
  katenary.v3/secrets: |-
    - PASSWORD
```


### katenary.v3/values

Environment variables to be added to the values.yaml

**Type**:  `[]string or map[string]string`

By default, all environment variables in the "env" and environment
files are added to configmaps with the static values set. This label
allows adding environment variables to the values.yaml file.

Note that the value inside the configmap is `{{ tpl vaname . }}`, so 
you can set the value to a template that will be rendered with the 
values.yaml file.

The value can be set with a documentation. This may help to understand 
the purpose of the variable.

**Example:**

```yaml
env:
  FOO: bar
  DB_NAME: mydb
  TO_CONFIGURE: something that can be changed in values.yaml
  A_COMPLEX_VALUE: example
labels:
  katenary.v3/values: |-
    # simple values, set as is in values.yaml
    - TO_CONFIGURE
    # complex values, set as a template in values.yaml with a documentation
    - A_COMPLEX_VALUE: |-
        This is the documentation for the variable to 
        configure in values.yaml.
        It can be, of course,  a multiline text.
```


### katenary.v3/values-from

Add values from another service.

**Type**:  `map[string]string`

This label allows adding values from another service to the current service.
It avoid duplicating values, environment or secrets that should be the same.

The key is the value to be added, and the value is the "key" to fetch in the
form `service_name.environment_name`.

**Example:**

```yaml
database:
  image: mariadb:10.5
  environment:
    MARIADB_USER: myuser
    MARIADB_PASSWORD: mypassword
  labels:
    # we can declare secrets
    katenary.v3/secrets: |-
      - MARIADB_PASSWORD
php:
  image: php:7.4-fpm
  environment:
    # it's duplicated in docker / podman
    DB_USER: myuser
    DB_PASSWORD: mypassword
  labels:
    # removes the duplicated, use the configMap and secrets from "database"
    katenary.v3/values-from: |-
      DB_USER: database.MARIADB_USER
      DB_PASSWORD: database.MARIADB_PASSWORD
```


<!-- STOP_DETAILED_DOC : do not remove this tag !-->
