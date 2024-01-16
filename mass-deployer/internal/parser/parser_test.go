package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	t.Run("invaild yaml format", func(t *testing.T) {
		conf := ` {
  "node_groups": [
    {
      "name": "group_b",
      "free_mru": 16384,
      "nodes_count": 10
    }
  ],
  "mnemonic": "example-mnemonic",
  "network": "dev"
} `
		_, err := ParseConfig([]byte(conf))
		assert.Error(t, err)
	})

	t.Run("invaild node_group", func(t *testing.T) {
		conf := `
vms:
  - name: example-vm
    vms_count: 2
    node_group: example-group
ssh_keys: 
  example1: ssh-key1
mnemonic: example-mnemonic
network: dev
    `
		_, err := ParseConfig([]byte(conf))
		assert.Error(t, err)
	})

	t.Run("invaild vms", func(t *testing.T) {
		conf := `
node_groups:
  - name: example-group
    nodes_count: 3
    free_cpu: 8
    free_mru: 16384
ssh_keys: 
  example1: ssh-key1
mnemonic: example-mnemonic
network: dev
`
		_, err := ParseConfig([]byte(conf))
		assert.Error(t, err)
	})

	t.Run("invaild ssh_keys", func(t *testing.T) {
		conf := `
node_groups:
  - name: example-group
    nodes_count: 3
    free_cpu: 8
    free_mru: 16384
vms:
  - name: example-vm
    vms_count: 2
    node_group: example-group
    cpu: 2
    mem: 4096
    flist: example-flist
    root_size: 20
    entry_point: /sbin/zinit init
mnemonic: example-mnemonic
network: dev
    `
		_, err := ParseConfig([]byte(conf))
		assert.Error(t, err)
	})

	t.Run("invaild mnemonic", func(t *testing.T) {
		conf := `
node_groups:
  - name: example-group
    nodes_count: 3
    free_cpu: 8
    free_mru: 16384
vms:
  - name: example-vm
    vms_count: 2
    node_group: example-group
    cpu: 2
    mem: 4096
    flist: example-flist
    root_size: 20
    entry_point: /sbin/zinit init
ssh_keys: 
  example1: ssh-key1
network: dev
    `
		_, err := ParseConfig([]byte(conf))
		assert.Error(t, err)
	})

	t.Run("invaild network", func(t *testing.T) {
		conf := `
node_groups:
  - name: example-group
    nodes_count: 3
    free_cpu: 8
    free_mru: 16384
vms:
  - name: example-vm
    vms_count: 2
    node_group: example-group
    cpu: 2
    mem: 4096
    flist: example-flist
    root_size: 20
    entry_point: /sbin/zinit init
ssh_keys: 
  example1: ssh-key1
network: example-network
mnemonic: example-mnemonic
    `
		_, err := ParseConfig([]byte(conf))
		assert.Error(t, err)
	})

	t.Run("valid config", func(t *testing.T) {
		conf := `
node_groups:
  - name: example-group
    nodes_count: 3
    free_cpu: 8
    free_mru: 16384
vms:
  - name: example-vm
    vms_count: 2
    node_group: example-group
    cpu: 2
    mem: 4096
    flist: example-flist
    root_size: 20
    entry_point: /sbin/zinit init
    ssh_key: example1
ssh_keys: 
  example1: ssh-key1
network: dev
mnemonic: example-mnemonic
    `
		_, err := ParseConfig([]byte(conf))
		assert.NoError(t, err)
	})
}
