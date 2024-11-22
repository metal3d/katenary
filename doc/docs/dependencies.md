# Why those dependencies?

Katenary uses `compose-go` and several Kubernetes official packages.

- `github.com/compose-spec/compose-go`: to parse compose files. It ensures :
    - that the project respects the "compose" specification
    - that Katenary uses the "compose" struct exactly the same way `podman compose` or `docker copose` does
- `github.com/spf13/cobra`: to parse command line arguments, sub-commands and flags. It also generates completion for
  bash, zsh, fish and PowerShell.
- `github.com/thediveo/netdb`: to get the standard names of a service from its port number
- `gopkg.in/yaml.v3`:
    - to generate `Chart.yaml` and `values.yaml` files (only)
    - to parse Katenary labels in the compose file
- `k8s.io/api` and `k8s.io/apimachinery` to create Kubernetes objects
- `sigs.k8s.io/yaml`: to generate Katenary YAML files in the format of Kubernetes objects

There are also some other packages used in the project, like `gopkg.in/yaml` to parse labels. I'm sorry to not list the
entire dependencies. You can check the `go.mod` file to see all the dependencies.
