# godot

A Dagger module for exporting [Godot Engine](https://godotengine.org/) projects. It downloads the Godot headless binary and export templates for the requested version, then runs a release or debug export using a named preset from `export_presets.cfg`.

## use

Export a Godot project using the "Linux/X11" preset:

```sh
dagger api call -m github.com/frantjc/daggerverse/godot --src . export-release --preset "Linux/X11" --path mygame binary export --path mygame
```

The preset name must exactly match an entry in your project's `export_presets.cfg`. Available presets are listed in the error message if the name is not found.


And run it:

```sh
> ./mygame
```
