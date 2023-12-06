# cronjobs

A Helm chart for cronjobs

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
# Standard Helm install
$ helm install  my-release cronjobs

# To use a custom namespace and force the creation of the namespace
$ helm install my-release --namespace my-namespace --create-namespace cronjobs

# To use a custom values file
$ helm install my-release -f my-values.yaml cronjobs
```

See the [Helm documentation](https://helm.sh/docs/intro/using_helm/) for more information on installing and managing the chart.

## Configuration

The following table lists the configurable parameters of the cronjobs chart and their default values.

| Parameter                           | Default        |
| ----------------------------------- | -------------- |
| `app.imagePullPolicy`               | `IfNotPresent` |
| `app.replicas`                      | `1`            |
| `app.repository.image`              | `nginx`        |
| `app.repository.tag`                | ``             |
| `backup.cronjob.imagePullPolicy`    | `IfNotPresent` |
| `backup.cronjob.repository.image`   | `alpine`       |
| `backup.cronjob.repository.tag`     | `1`            |
| `backup.cronjob.schedule`           | `@hourly`      |
| `backup.imagePullPolicy`            | `IfNotPresent` |
| `backup.replicas`                   | `1`            |
| `backup.repository.image`           | `alpine`       |
| `backup.repository.tag`             | `1`            |
| `withrbac.cronjob.imagePullPolicy`  | `IfNotPresent` |
| `withrbac.cronjob.repository.image` | `busybox`      |
| `withrbac.cronjob.repository.tag`   | ``             |
| `withrbac.cronjob.schedule`         | `@daily`       |
| `withrbac.imagePullPolicy`          | `IfNotPresent` |
| `withrbac.replicas`                 | `1`            |
| `withrbac.repository.image`         | `busybox`      |
| `withrbac.repository.tag`           | ``             |


