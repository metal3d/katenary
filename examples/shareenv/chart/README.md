# shareenv

A Helm chart for shareenv

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
# Standard Helm install
$ helm install  my-release shareenv

# To use a custom namespace and force the creation of the namespace
$ helm install my-release --namespace my-namespace --create-namespace shareenv

# To use a custom values file
$ helm install my-release -f my-values.yaml shareenv
```

See the [Helm documentation](https://helm.sh/docs/intro/using_helm/) for more information on installing and managing the chart.

## Configuration

The following table lists the configurable parameters of the shareenv chart and their default values.

| Parameter               | Default        |
| ----------------------- | -------------- |
| `app1.imagePullPolicy`  | `IfNotPresent` |
| `app1.replicas`         | `1`            |
| `app1.repository.image` | `nginx`        |
| `app1.repository.tag`   | `1`            |
| `app2.imagePullPolicy`  | `IfNotPresent` |
| `app2.replicas`         | `1`            |
| `app2.repository.image` | `nginx`        |
| `app2.repository.tag`   | `1`            |


