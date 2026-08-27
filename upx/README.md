# upx

A Dagger module for compressing executables using [UPX](https://upx.github.io/).

## use

Compress a binary:

```sh
dagger api call -m github.com/frantjc/daggerverse/upx pack --executable ./myapp export --path ./myapp
```

Pass `--lzma` for better compression at the cost of decompression speed, or `--brute` to try all methods and pick the smallest result.
