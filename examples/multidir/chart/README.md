# multidir

A Helm chart for multidir

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
# Standard Helm install
$ helm install  my-release multidir

# To use a custom namespace and force the creation of the namespace
$ helm install my-release --namespace my-namespace --create-namespace multidir

# To use a custom values file
$ helm install my-release -f my-values.yaml multidir
```

See the [Helm documentation](https://helm.sh/docs/intro/using_helm/) for more information on installing and managing the chart.

## Configuration

The following table lists the configurable parameters of the multidir chart and their default values.

| Parameter              | Default        |
| ---------------------- | -------------- |
| `bar.imagePullPolicy`  | `IfNotPresent` |
| `bar.replicas`         | `1`            |
| `bar.repository.image` | `alpine`       |
| `bar.repository.tag`   | ``             |
| `foo.imagePullPolicy`  | `IfNotPresent` |
| `foo.replicas`         | `1`            |
| `foo.repository.image` | `alpine`       |
| `foo.repository.tag`   | ``             |


