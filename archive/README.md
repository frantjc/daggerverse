# archive

A Dagger module for creating and extracting archive files. Supports tar (with optional gzip compression), tgz, tar.xz, and zip formats.

## use

Create a `.tgz` from a directory:

```sh
dagger call -m github.com/frantjc/daggerverse/archive tar --directory ./dist --gzip export --path dist.tgz
```

Unpack a tarball back to a directory:

```sh
dagger call -m github.com/frantjc/daggerverse/archive untar --file dist.tgz export --path ./dist
```

The compression format for `untar` is detected automatically from the file extension (`.tgz`, `.tar.gz`, `.tar.xz`).
