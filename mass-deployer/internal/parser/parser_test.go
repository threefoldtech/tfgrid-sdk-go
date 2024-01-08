package parser

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("invaild file name", func(t *testing.T) {
		_, err := ParseConfig("config")
		assert.Error(t, err)
	})

	t.Run("invaild file extension", func(t *testing.T) {
		_, err := ParseConfig("config.md")
		assert.Error(t, err)
	})

	t.Run("not found file", func(t *testing.T) {
		testFile := path.Join(tempDir, "conf.json")

		_, err := ParseConfig(testFile)
		assert.Error(t, err)
	})

	t.Run("yaml file", func(t *testing.T) {
		configFile := path.Join(tempDir, "conf.yaml")
		_, err := os.Create(configFile)
		if !assert.NoError(t, err) {
			return
		}
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
sshkey: example-ssh-key
mnemonic: example-mnemonic
network: example-network
    `
		err = os.WriteFile(configFile, []byte(conf), 0667)
		if !assert.NoError(t, err, "failed to write to test file") {
			return
		}

		_, err = ParseConfig(configFile)
		assert.NoError(t, err)
	})

	t.Run("json file", func(t *testing.T) {
		configFile := path.Join(tempDir, "conf.json")
		_, err := os.Create(configFile)
		if !assert.NoError(t, err) {
			return
		}
		conf := `
{
  "node_groups": [
    {
      "name": "group_a",
      "nodes_count": 3,
      "free_cpu": 2,
      "free_mru": 16384,
      "free_ssd": 524288000
    }
  ],
  "vms": [
    {
      "name": "examplevm",
      "vms_count": 5,
      "node_group": "group_a",
      "cpu": 1,
      "mem": 256,
      "flist": "example-flist",
      "root_size": 0,
      "entry_point": "/sbin/zinit init"
    }
  ],
  "sshkey": "example-ssh-key",
  "mnemonic": "example-mnemonic",
  "network": "example-network"
}
    `
		err = os.WriteFile(configFile, []byte(conf), 0667)
		if !assert.NoError(t, err, "failed to write to test file") {
			return
		}

		_, err = ParseConfig(configFile)
		assert.NoError(t, err)
	})
}
