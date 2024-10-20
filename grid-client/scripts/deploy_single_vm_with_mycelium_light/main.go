package main

import (
	"context"
	"errors"
	"flag"
	"net"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func main() {
	ctx := context.Background()
	tf, publicKey, err := setup()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	nodeID, err := filterNode(tf)
	if err != nil {
		log.Fatal().Err(err).Msg("no available nodes found")
	}

	myceliumKey, err := workloads.RandomMyceliumKey()
	if err != nil {
		log.Debug().Err(err).Send()
	}

	network := workloads.ZNetLight{
		Name:        "test_net_light",
		Description: "network to deploy vm with mycelium",
		Nodes:       []uint32{nodeID},
		IPRange: zos.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		MyceliumKeys: map[uint32][]byte{nodeID: myceliumKey},
	}

	err = tf.NetworkDeployer.Deploy(ctx, &network)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	myceliumSeed, err := workloads.RandomMyceliumIPSeed()
	if err != nil {
		log.Debug().Err(err).Send()
	}

	vm := workloads.VMLight{
		Name:           "vm",
		NodeID:         nodeID,
		NetworkName:    network.Name,
		CPU:            1,
		MemoryMB:       256,
		RootfsSizeMB:   10 * 1024,
		Flist:          "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		Entrypoint:     "/sbin/zinit init",
		IP:             "10.1.2.5",
		MyceliumIPSeed: myceliumSeed,
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
	}

	dl := workloads.NewDeployment("vm_with_mycelium", nodeID, "", nil, network.Name, nil, nil, nil, []workloads.VMLight{vm}, nil, nil)
	err = tf.DeploymentDeployer.Deploy(context.Background(), &dl)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	dl, err = tf.State.LoadDeploymentFromGrid(ctx, nodeID, dl.Name)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Info().Str("mycelium ip", dl.VmsLight[0].MyceliumIP).Send()
}

func setup() (deployer.TFPluginClient, string, error) {
	mnemonic := os.Getenv("MNEMONIC")
	log.Debug().Str("MNEMONIC", mnemonic).Send()

	n := os.Getenv("NETWORK")
	log.Debug().Str("NETWORK", n).Send()

	var publicKeyPath string
	flag.StringVar(&publicKeyPath, "ssh-key", "", "path to user ssh key")
	flag.Parse()
	if publicKeyPath == "" {
		return deployer.TFPluginClient{}, "", errors.New("path to ssh key should not be empty")
	}

	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return deployer.TFPluginClient{}, "", err
	}

	tf, err := deployer.NewTFPluginClient(mnemonic, deployer.WithNetwork(n))
	if err != nil {
		return deployer.TFPluginClient{}, "", err
	}
	return tf, string(publicKey), nil
}

func filterNode(tf deployer.TFPluginClient) (uint32, error) {
	f := types.NodeFilter{Status: []string{"up"}, Features: []string{zos.NetworkLightType, zos.ZMachineLightType}}
	nodes, err := deployer.FilterNodes(context.Background(), tf, f, nil, nil, []uint64{convertGBToBytes(10)}, 1)
	if err != nil {
		return 0, err
	}

	return uint32(nodes[0].NodeID), err
}

func convertGBToBytes(gb uint64) uint64 {
	bytes := gb * 1024 * 1024 * 1024
	return bytes
}
