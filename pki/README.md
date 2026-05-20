# pki

A Dagger module for managing a private PKI (Public Key Infrastructure) using OpenSSL. It can generate or reuse a CA key, issue TLS server certificates, and produce PKCS#12 code-signing bundles.
## use

Generate a CA and export a TLS certificate for `localhost`:

```sh
dagger call -m github.com/frantjc/daggerverse/pki ca tls-key-pair --hostname localhost crt export --path tls.crt
```

Omit `--ca-key` to generate a fresh RSA-4096 CA key. Pass `--ca-key` and `--ca-crt` to reuse an existing authority across pipeline runs.
