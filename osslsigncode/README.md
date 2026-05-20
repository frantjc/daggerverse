# osslsigncode

A Dagger module that builds [osslsigncode](https://github.com/mtrojnar/osslsigncode) from source The resulting binary can be consumed by [other modules](../godot/) that need to sign Windows binaries.

## use

Get the `osslsigncode` binary:

```sh
dagger call -m github.com/frantjc/daggerverse/osslsigncode binary export --path ./osslsigncode
```

Depending on if you have the dynamically-linked dependencies installed, you should be able to execute `osslsigncode` on a Linux machine.

```sh
> ./osslsigncode -v
./osslsigncode: /lib/x86_64-linux-gnu/libcrypto.so.3: version `OPENSSL_3.6.0' not found (required by ./osslsigncode)
```
