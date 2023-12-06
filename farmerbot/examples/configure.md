# How to create a config file for farmerbot

## YML

Create a new yml/yaml file `config.yml` and add your configurations:

```yml
farm_id: "<your farm ID, required>"
included_nodes:
  - "<your node ID to be included, required at least 2>"
excluded_nodes:
  - "<your node ID to be excluded, optional>"
never_shutdown_nodes:
  - "<your node ID to be never shutdown, optional>"
power:
  periodic_wake_up_start: "<daily time to wake up nodes for your farm, default is the time your run the command, format is 00:00AM or 00:00PM, optional>"
  wake_up_threshold: "<the threshold number for resources usage that will need another node to be on, default is 80, optional>"
  periodic_wake_up_limit: "<the number (limit) of nodes to be waken up everyday, default is 1, optional>"
  overprovision_cpu: "<how much node allows over provisioning the CPU , default is 1, range: [1;4], optional>"
```
