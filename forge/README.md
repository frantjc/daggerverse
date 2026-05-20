# forge

A Dagger module for [`forge`](https://github.com/frantjc/forge), a tool for running reusable [GitHub Actions](https://docs.github.com/en/actions) steps outside of GitHub. It sets up the action's runtime environment (workspace, env files, toolcache, an in-pipeline actions cache service) and executes the action's pre, main, and post steps inside a Dagger container.

## use

Run [`actions/setup-node`](https://github.com/actions/setup-node):

```sh
dagger call -m github.com/frantjc/daggerverse/forge use --action actions/setup-node@v4 main toolcache
```

Actions that run using `docker`, `node12`, `node16`, `node20`, and `node24` are supported.
