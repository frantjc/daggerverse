# gh

A Dagger module wrapping the [GitHub CLI](https://cli.github.com/) for use creating GitHub releases and upload release assets.

## use

Create a GitHub release:

```sh
dagger call -m github.com/frantjc/daggerverse/gh --github-token env://GITHUB_TOKEN release --repo owner/repo --tag v1.2.3 create --generate-notes --latest
```

`GITHUB_TOKEN` must have `contents: write` permission on the target repository.
