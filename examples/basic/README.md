# Basic example

This is a basic example of what can do Katenary with standard docker-compose file.

In this example:

- `depends_on` yield a `initContainer` in the webapp ddeployment to wait for database
- so we need to declare the listened port inside `database` container as we don't use it with docker-compose- also, we needed to declare that `DB_HOST` is actually a service name

Take a look on [chart/basic](chart/basic) directory to see what `katenary convert` command has generated.
