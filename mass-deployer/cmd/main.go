package cmd

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/internal/parser"
	deployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
)

func Execute() {
	var configFile string
	flag.StringVar(&configFile, "config", "", "path to config file")
	flag.Parse()

	if configFile == "" {
		log.Fatal("couldn't locate config file")
	}

	cfg, err := parser.ParseConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	d, err := deployer.NewDeployer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Minute)
	defer cancel()

	groupsNodes := map[string][]int{}

	for _, group := range cfg.NodeGroups {
		nodes, err := d.FilterNodes(group, ctx)
		if err != nil {
			log.Default().Println(err)
			continue
		}

		nodesIDs := []int{}
		for _, node := range nodes {
			nodesIDs = append(nodesIDs, node.NodeID)
		}
		groupsNodes[group.Name] = nodesIDs
	}
	vmsWorkloads := d.ParseVms(cfg.Vms)
	for group, vms := range vmsWorkloads {
		// deploy every group of vms sparatly as a mass deployment
	}
}
