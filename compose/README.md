# compose

A Dagger module that parses a `docker-compose.yml` and exposes its services as a Dagger `Service`. This lets you bring up a Compose-defined stack — including service dependencies, bind mounts, volumes, and environment variables — and bind it to other Dagger containers for integration testing.

## use

Start the services defined in your project's `docker-compose.yml`:

```sh
dagger call -m github.com/frantjc/daggerverse/compose --source . up
```

Only `ingress` port mode is supported. Services that declare `depends_on` are started in dependency order with the dependency bound as a service alias matching its name in the Compose file.
