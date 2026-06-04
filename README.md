# taskmaster

Supervisor-style job control. Spawns child processes from a YAML config, restarts them per policy, reloads on SIGHUP without disrupting unchanged programs. 42 school project.

## Requirements

Go 1.26+, and `nginx` on `$PATH` if you want to run `configs/subj.yaml`. The repo-root Nix flake provides both via `nix develop`.

## Build & run

```sh
make build      # ./taskmaster
make run        # ./taskmaster -c configs/subj.yaml
make test
```

## Shell

```
status
start    <name> [index]
stop     <name> [index]
restart  <name> [index]
reload
shutdown
```

## Signals

`SIGTERM` / `SIGINT` — graceful shutdown: configured stopsignal first, then SIGKILL after `stoptime`.
`SIGHUP` — re-read config and apply the diff.

## Reload

Specs are compared field-by-field:

- unchanged → keep PIDs
- numprocs-only → resize; survivors keep their PIDs
- any other field → restart the affected procs
- program added / removed → spawn / stop accordingly
