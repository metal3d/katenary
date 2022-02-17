# Make it possible to bind several containers in one pod

In this example, we need to make nginx and php-fpm to run inside the same "pod". The reason is that we configured FPM to listen an unix socket instead of the 9000 port.

Because NGinx will need to connect to the unix socket wich is a file, both containers should share the same node and work together.

So, in the docker-compose file, we need to declare:
- `katenary.io/empty-dirs: socket` where `socket` is the "volume name", this will avoid the creation of a PVC
- `katenary.io/same-pod: http` in `php` container to declare that this will be added in the `containers` section of the `http` deployment

You can note that we also use `configmap-volumes` to declare our configuration as `configMap`.

Take a look on [chart/same-pod](chart/same-pod) directory to see the result of the `katenary convert` command.
