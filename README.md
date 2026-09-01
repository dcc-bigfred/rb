# rb

CLI for **RailBOX RB23xx** decoder sound slots over the decoder's Soft-AP
HTTP interface (`http://192.168.4.1`). Join the locomotive Wi-Fi before
running commands.

## Build

```bash
make build    # bin/rb
make test
```

## Sound slots

```bash
# Remove every file in slot 1
rb sound clear 1

# Upload / delete so a local directory matches slot 1
rb sound sync 1 ./sounds

# Preview without writing
rb sound sync 1 ./sounds --dry-run

# Re-sync whenever files in the directory change
rb sound sync 1 ./sounds --watch
```

By default `sync` also re-uploads the 5 files modified in the last 24 hours.
Pass `--without-last` to disable that.

HTTP timeout defaults to 10 seconds (`--timeout`).
