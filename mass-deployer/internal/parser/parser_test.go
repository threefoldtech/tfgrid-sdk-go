package parser

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := path.Join(tempDir, "conig.yaml")
	t.Run("invaild yaml format", func(t *testing.T) {
		conf := ` {
  "node_groups": [
    {
      "nodes_count": 10
    }
} `
		err := os.WriteFile(configPath, []byte(conf), 0667)
		assert.NoError(t, err)

		configFile, err := os.Open(configPath)
		assert.NoError(t, err)

		_, err = ParseConfig(configFile)
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
network: dev
mnemonic: rival oyster defense garbage fame disease mask mail family wire village vibrant index fuel dolphin
    `
		err := os.WriteFile(configPath, []byte(conf), 0667)
		assert.NoError(t, err)

		configFile, err := os.Open(configPath)
		assert.NoError(t, err)

		_, err = ParseConfig(configFile)
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
network: dev
mnemonic: rival oyster defense garbage fame disease mask mail family wire village vibrant index fuel dolphin
`
		err := os.WriteFile(configPath, []byte(conf), 0667)
		assert.NoError(t, err)

		configFile, err := os.Open(configPath)
		assert.NoError(t, err)

		_, err = ParseConfig(configFile)
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
    flist: https://hub.grid.tf/tf-official-apps/base:latest.flist
    root_size: 20
    entry_point: /sbin/zinit init
network: dev
mnemonic: rival oyster defense garbage fame disease mask mail family wire village vibrant index fuel dolphin
    `
		err := os.WriteFile(configPath, []byte(conf), 0667)
		assert.NoError(t, err)

		configFile, err := os.Open(configPath)
		assert.NoError(t, err)

		_, err = ParseConfig(configFile)
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
    flist: https://hub.grid.tf/tf-official-apps/base:latest.flist
    root_size: 20
    entry_point: /sbin/zinit init
ssh_keys: 
  example1: ssh-key1
network: dev
    `
		err := os.WriteFile(configPath, []byte(conf), 0667)
		assert.NoError(t, err)

		configFile, err := os.Open(configPath)
		assert.NoError(t, err)

		_, err = ParseConfig(configFile)
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
    flist: https://hub.grid.tf/tf-official-apps/base:latest.flist
    root_size: 20
    entry_point: /sbin/zinit init
ssh_keys: 
  example1: ssh-key1
network: example-network
mnemonic: example-mnemonic
    `
		err := os.WriteFile(configPath, []byte(conf), 0667)
		assert.NoError(t, err)

		configFile, err := os.Open(configPath)
		assert.NoError(t, err)

		_, err = ParseConfig(configFile)
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
    flist: https://hub.grid.tf/tf-official-apps/base:latest.flist
    root_size: 20
    entry_point: /sbin/zinit init
    ssh_key: example1
ssh_keys: 
  example1: ssh-key1
network: dev
mnemonic: rival oyster defense garbage fame disease mask mail family wire village vibrant index fuel dolphin
    `
		err := os.WriteFile(configPath, []byte(conf), 0667)
		assert.NoError(t, err)

		configFile, err := os.Open(configPath)
		assert.NoError(t, err)

		_, err = ParseConfig(configFile)
		assert.NoError(t, err)
	})
}
