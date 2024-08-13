# Grid-Compose

is a tool for running multi-vm applications on TFGrid defined using a Yaml formatted file.

## Usage

`REQUIRED` EnvVars:

- `MNEMONIC`: your secret words
- `NETWORK`: one of (dev, qa, test, main)

```bash
grid-compose [OPTIONS] [COMMAND]

OPTIONS:
    -f path to yaml file, default is ./grid-compose.yaml

COMMANDS:
    - version: shows the project version
    - up:      deploy the app
    - down:    cancel all deployments
    - ps: list deployments on the grid
```

Run:

```bash
make build
```

Then:

```bash
./bin/grid-compose [COMMAND]
```

For example:

```bash
./bin/grid-compose ps -f example/multiple_services_diff_network_3.yml
```
