# Grid-Compose

is a tool for running multi-vm applications on TFGrid defined using a Yaml formatted file.

## Usage

`REQUIRED` EnvVars:

- `MNEMONIC`: your secret words
- `NETWORK`: one of (dev, qa, test, main)

```bash
grid-compose [OPTIONS] [COMMAND]

OPTIONS:
    -f, --file: path to yaml file, default is ./grid-compose.yaml

COMMANDS:
    - version: shows the project version
    - up:      deploy the app
    - down:    cancel all deployments
    - ps: list deployments on the grid
        OPTIONS:
            - -v, --verbose: show full details of each deployment
            - -o, --output: redirects the output to a file given its path
```

Export env vars using:

```bash
export MNEMONIC=your_mnemonics
export NETWORK=working_network
```

Run:

```bash
make build
```

To use any of the commands, run:

```bash
./bin/grid-compose [COMMAND]
```

For example:

```bash
./bin/grid-compose ps -f example/multiple_services_diff_network_3.yml
```

## Usage For Each Command

### up

The up command deploys the services defined in the yaml file to the grid.

Refer to the [cases](docs/cases.md) for more information on the cases supported.

Refer to examples in the [examples](examples) directory to have a look at different possible configurations.

```bash
./bin/grid-compose up [OPTIONS]
```

OPTIONS:

- `-f, --file`: path to the yaml file, default is `./grid-compose.yaml`

### down

```bash
./bin/grid-compose down [OPTIONS]
```

OPTIONS:

- `-f, --file`: path to the yaml file, default is `./grid-compose.yaml`

### ps

```bash
./bin/grid-compose ps [FLAGS] [OPTIONS]
```

OPTIONS:

- `-f, --file`: path to the yaml file, default is `./grid-compose.yaml`
- `-v, --verbose`: show full details of each deployment
- `-o, --output`: redirects the output to a file given its path(in json format)
