# Grid-Compose

is a tool similar to docker-compose created for running multi-vm applications on TFGrid defined using a Yaml formatted file.

The yaml file's structure is defined in [docs/config](docs/config.md).

## Usage

`REQUIRED` EnvVars:

- `MNEMONIC`: your secret words
- `NETWORK`: one of (dev, qa, test, main)

```bash
grid-compose [OPTIONS] [COMMAND]

OPTIONS:
    -f, --file: path to yaml file, default is ./grid-compose.yml

COMMANDS:
    - version: shows the project version
    - up:      deploy the app
    - down:    cancel all deployments
    - ps: list deployments on the grid
        OPTIONS:
            - -v, --verbose: show full details of each deployment
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

- `-f, --file`: path to the yaml file, default is `./grid-compose.yml`

### Example

```bash
./bin/grid-compose up
```

output:

```bash
3:40AM INF starting peer session=tf-848216 twin=8658
3:40AM INF deploying network... name=miaminet node_id=14
3:41AM INF deployed successfully
3:41AM INF deploying vm... name=database node_id=14
3:41AM INF deployed successfully
3:41AM INF deploying network... name=miaminet node_id=14
3:41AM INF deployed successfully
3:41AM INF deploying vm... name=server node_id=14
3:41AM INF deployed successfully
3:41AM INF all deployments deployed successfully
```

### down

The down command cancels all deployments on the grid.

```bash
./bin/grid-compose down [OPTIONS]
```

OPTIONS:

- `-f, --file`: path to the yaml file, default is `./grid-compose.yml`

### Example

```bash
./bin/grid-compose down
```

output:

```bash
3:45AM INF starting peer session=tf-854215 twin=8658
3:45AM INF canceling deployments projectName=vm/compose/8658/net1
3:45AM INF canceling contracts project name=vm/compose/8658/net1
3:45AM INF project is canceled project name=vm/compose/8658/net1
```

### ps

The ps command lists all deployments on the grid.

```bash
./bin/grid-compose ps [FLAGS] [OPTIONS]
```

OPTIONS:

- `-f, --file`: path to the yaml file, default is `./grid-compose.yml`

### Example

```bash
./bin/grid-compose ps
```

output:

```bash
3:43AM INF starting peer session=tf-851312 twin=8658

Deployment Name | Node ID         | Network         | Services        | Storage         | State      | IP Address
------------------------------------------------------------------------------------------------------------------------------------------------------
dl_database     | 14              | miaminet        | database        | dbdata          | ok         | wireguard: 10.20.2.2
dl_server       | 14              | miaminet        | server          | webdata         | ok         | wireguard: 10.20.2.3
```

FLAGS:

- `-v, --verbose`: show full details of each deployment

### version

The version command shows the project's current version.

```bash
./bin/grid-compose version
```

## Future Work

Refer to [docs/future_work.md](docs/future_work.md) for more information on the future work that is to be done on the grid-compose project.
