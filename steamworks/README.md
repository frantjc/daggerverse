# Steamworks

A Dagger module for Steamworks tooling.

## use

Call functions on `steamcmd`. Without `--password` it logs in as `anonymous`, which is not enough for commands that need a real account, such as `app-build` and `drm-wrap`. Accounts protected by Steam Guard must also pass `--steam-guard-code`.

Upload a directory to Steam with `steamcmd`'s `+run_app_build` and return the build output:

```sh
dagger call steamcmd app-build \
  --username "$STEAM_USERNAME" \
  --password env://STEAM_PASSWORD \
  app-build \
  --app-id 480 \
  --content ./build \
  --depots '[{"depotId": 481, "recursive": true}]' \
  export --path ./output
```

Wrap an executable with [Steam DRM](https://partner.steamgames.com/doc/features/drm):

```sh
dagger call steamcmd drm-wrap \
  --username "$STEAM_USERNAME" \
  --password env://STEAM_PASSWORD \
  drm-wrap \
  --app-id 480 \
  --executable ./game.exe \
  export --path ./game.wrapped.exe
```

Pass `--compatibility` to disable obfuscation and `--skip-debugger-check` to skip debugger checks.
