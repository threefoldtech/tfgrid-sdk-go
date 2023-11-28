# How to create a config file for farmerbot

## JSON

Create a new json file `config.json` and add your configurations:

```json
"farm_id": "<your farm ID, required>",
"included_nodes": ["<your node ID to be included, required at least 2>"],
"excluded_nodes": ["<your node ID to be excluded, optional>"],
"power": {
    "wake_up_threshold": "<the threshold for resources usage that will need another node to be on, default is 80, optional>",
    "periodic_wake_up_start": "<daily time to wake up nodes for your farm, default is the time your run the command, format is 00:00AM or 00:00PM, optional>",
    "periodic_wake_up_limit": "<the number (limit) of nodes to be waken up everyday, default is 1, optional>",
    "overprovision_cpu": "<how much node allows over provisioning the CPU , default is 1, range: [1;4], optional>"
}
```

## YML

Create a new yml/yaml file `config.yml` and add your configurations:

```yml
farm_id: "<your farm ID, required>"
included_nodes:
  - "<your node ID to be included, required at least 2>"
excluded_nodes:
  - "<your node ID to be excluded, optional>"
power:
  periodic_wake_up_start: "<daily time to wake up nodes for your farm, default is the time your run the command, format is 00:00AM or 00:00PM, optional>"
  wake_up_threshold: "<the threshold number for resources usage that will need another node to be on, default is 80, optional>"
  periodic_wake_up_limit: "<the number (limit) of nodes to be waken up everyday, default is 1, optional>"
  overprovision_cpu: "<how much node allows over provisioning the CPU , default is 1, range: [1;4], optional>"
```

## TOML

Create a new toml file `config.toml` and add your configurations:

```toml
farm_id = "<your farm ID, required>"
included_nodes = ["<your node ID to be included, required at least 2>"]
excluded_nodes = ["<your node ID to be excluded, optional>"]

[power]
periodic_wake_up_start = "<daily time to wake up nodes for your farm, default is the time your run the command, format is 00:00AM or 00:00PM, optional>"
wake_up_threshold = "<the threshold number for resources usage that will need another node to be on, default is 80, optional>"
periodic_wake_up_limit = "<the number (limit) of nodes to be waken up everyday, default is 1, optional>"
overprovision_cpu = "<how much node allows over provisioning the CPU , default is 1, range: [1;4], optional>"
```
