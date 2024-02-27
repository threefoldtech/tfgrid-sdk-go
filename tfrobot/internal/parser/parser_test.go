package parser

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	tfrobot "github.com/threefoldtech/tfgrid-sdk-go/tfrobot/pkg/deployer"
	"gopkg.in/yaml.v3"
)

func TestParseConfig(t *testing.T) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	validFlist := "https://hub.grid.tf/tf-official-apps/base:latest.flist"

	confStruct := tfrobot.Config{
		NodeGroups: []tfrobot.NodesGroup{
			{
				Name:       "group_a",
				NodesCount: 5,
				FreeCPU:    2,
				FreeMRU:    1,
				FreeSRU:    50,
				FreeHRU:    50,
				PublicIP4:  true,
				Region:     "Europe",
			},
		},
		Vms: []tfrobot.Vms{
			{
				Name:       "examplevm",
				Count:      4,
				NodeGroup:  "group_a",
				FreeCPU:    2,
				FreeMRU:    1,
				PublicIP4:  true,
				Flist:      validFlist,
				Entrypoint: "/sbin/zinit init",
				SSHKey:     "example1",
			},
		},
		SSHKeys: map[string]string{
			"example1": "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAklOUpkDHrfHY17SbrmTIpNLTGK9Tjom/BWDSUGPl+nafzlHDTYW7hdI4yZ5ew18JH4JW9jbhUFrviQzM7xlELEVf4h9lFX5QVkbPppSwg0cda3Pbv7kOdJ/MTyBlWXFCR+HAo3FXRitBqxiX1nKhXpHAZsMciLq8V6RjsNAQwdsdMFvSlVK/7XAt3FaoJoAsncM1Q9x5+3V0Ww68/eIFmb1zuUFljQJKprrX88XypNDvjYNby6vw/Pb0rwert/EnmZ+AW4OZPnTPI89ZPmVMLuayrD2cE86Z/il8b+gw3r3+1nKatmIkjn2so1d01QraTlMqVSsbxNrRFi9wrf+M7Q== schacon@mylaptop.local",
		},
		Mnemonic: "rival oyster defense garbage fame disease mask mail family wire village vibrant index fuel dolphin",
		Network:  "dev",
	}

	opts := []deployer.PluginOpt{
		deployer.WithRMBTimeout(30),
		deployer.WithRMBInMemCache(),
	}

	tfpluginClient, err := deployer.NewTFPluginClient(confStruct.Mnemonic, peer.KeyTypeSr25519, confStruct.Network, opts...)
	assert.NoError(t, err)

	t.Run("invalid yaml format", func(t *testing.T) {
		conf := ` {
  "node_groups": [
    {
      "nodes_count": 10
    }
} `
		configFile := strings.NewReader(conf)

		_, err := ParseConfig(configFile, false)
		assert.Error(t, err)
	})

	t.Run("invalid node_group", func(t *testing.T) {
		conf := confStruct
		conf.NodeGroups = []tfrobot.NodesGroup{}

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("invalid vms", func(t *testing.T) {
		conf := confStruct
		conf.Vms = []tfrobot.Vms{}

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("invalid ssh_keys", func(t *testing.T) {
		conf := confStruct
		conf.SSHKeys = map[string]string{}

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("invalid mnemonic", func(t *testing.T) {
		conf := confStruct
		conf.Mnemonic = "mnemonic"

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		configFile := strings.NewReader(string(data))

		_, err = ParseConfig(configFile, false)
		assert.Error(t, err)
	})

	t.Run("invalid network", func(t *testing.T) {
		conf := confStruct
		conf.Network = "network"

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		configFile := strings.NewReader(string(data))

		_, err = ParseConfig(configFile, false)
		assert.Error(t, err)
	})

	t.Run("empty flist", func(t *testing.T) {
		conf := confStruct
		conf.Vms[0].Flist = ""

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		conf.Vms[0].Flist = validFlist

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("invalid flist extension", func(t *testing.T) {
		conf := confStruct
		conf.Vms[0].Flist = "https://example-list.list"

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		conf.Vms[0].Flist = validFlist

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("invalid md5", func(t *testing.T) {
		conf := confStruct
		conf.Vms[0].Flist = "https://example-flist.flist"

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		conf.Vms[0].Flist = validFlist

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("zero cpu in node group", func(t *testing.T) {
		conf := confStruct
		conf.NodeGroups[0].FreeCPU = 0

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		conf.NodeGroups[0].FreeCPU = 2

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("zero memory in node group", func(t *testing.T) {
		conf := confStruct
		conf.NodeGroups[0].FreeMRU = 0

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		conf.NodeGroups[0].FreeMRU = 1

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("zero cpu in vm", func(t *testing.T) {
		conf := confStruct
		conf.Vms[0].FreeCPU = 0

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		conf.Vms[0].FreeCPU = 2

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("zero memory in vm", func(t *testing.T) {
		conf := confStruct
		conf.Vms[0].FreeMRU = 0

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		conf.Vms[0].FreeMRU = 1

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("cpu exceed limit in vm", func(t *testing.T) {
		conf := confStruct
		conf.Vms[0].FreeCPU = 35

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		conf.Vms[0].FreeCPU = 2

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("memory exceed limit in vm", func(t *testing.T) {
		conf := confStruct
		conf.Vms[0].FreeMRU = 300

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		conf.Vms[0].FreeMRU = 1

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("root size exceed limit in vm", func(t *testing.T) {
		conf := confStruct
		conf.Vms[0].RootSize = 20000

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		conf.Vms[0].RootSize = 0

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.Error(t, err)
	})

	t.Run("valid config", func(t *testing.T) {
		conf := confStruct

		data, err := yaml.Marshal(conf)
		assert.NoError(t, err)

		configFile := strings.NewReader(string(data))

		cfg, err := ParseConfig(configFile, false)
		assert.NoError(t, err)

		err = ValidateConfig(cfg, tfpluginClient)
		assert.NoError(t, err)
	})

	t.Run("valid json config", func(t *testing.T) {
		conf := confStruct

		data, err := json.Marshal(conf)
		assert.NoError(t, err)

		configFile := strings.NewReader(string(data))

		parsedConf, err := ParseConfig(configFile, true)
		assert.NoError(t, err)
		assert.Equal(t, conf, parsedConf)

		err = ValidateConfig(parsedConf, tfpluginClient)
		assert.NoError(t, err)
	})
}
