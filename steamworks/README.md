# Steamworks

A Dagger module for Steamworks tooling.

## use

Call functions on `steamcmd`. Without `--password` it logs in as `anonymous`, which is not enough for commands that need a real account, such as `app-build` and `drm-wrap`.

Accounts protected by Steam Guard can authenticate with any of the following (as in [game-ci/steam-deploy](https://github.com/game-ci/steam-deploy)):

- `--shared-secret`: the base64 `shared_secret` of the account's Steam Guard mobile authenticator. A fresh code (TOTP) is generated each time `steamcmd` is constructed.
- `--steam-guard-code`: a code you generated yourself. Takes precedence over `--shared-secret`.
- `--config-vdf`: the contents of the `config.vdf` (e.g. `file://config.vdf`) of a session previously authorized by logging in with `steamcmd` locally, so that no code is needed at all. Still requires `--password`.

```sh
dagger call steamcmd \
  --username "$STEAM_USERNAME" \
  --password env://STEAM_PASSWORD \
  --shared-secret env://STEAM_SHARED_SECRET \
  --config-vdf env://STEAM_CONFIG_VDF \
  ...
```

Upload a directory to Steam with `steamcmd`'s `+run_app_build`. The build output directory is backed by a cache volume per app ID, so steamcmd's build cache is reused between builds:

```sh
dagger call steamcmd \
  --username "$STEAM_USERNAME" \
  --password env://STEAM_PASSWORD \
  app-build \
  --app-id 480 \
  --content ./build \
  --depots '[{"depotId": 481, "recursive": true}]'
```

By default the uploaded build is not set live on any branch. Pass `--branch` to set it live on a beta branch once uploaded (Steam does not allow this for the `default` branch):

```sh
dagger call steamcmd \
  --username "$STEAM_USERNAME" \
  --password env://STEAM_PASSWORD \
  app-build \
  --app-id 480 \
  --content ./build \
  --depots '[{"depotId": 481, "recursive": true}]' \
  --branch beta
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
