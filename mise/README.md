# mise

A Dagger module that installs and activates tools managed by [mise](https://mise.jdx.dev/). Given a source directory or a `mise.toml` config file, it installs the declared tool versions so subsequent pipeline steps can use them without any host dependencies.

## use

Install tools from a project's `mise.toml` and open a terminal:

```sh
dagger api call -m github.com/frantjc/daggerverse/mise container terminal
```
