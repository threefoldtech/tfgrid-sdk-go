package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/internal/parser"
	deployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
)

func Execute() {
	var configFile string
	flag.StringVar(&configFile, "c", "", "path to config file")
	flag.Parse()

	if configFile == "" {
		log.Fatal("couldn't locate config file")
	}

	cfg, err := parser.ParseConfig(configFile)
	if err != nil {
		log.Fatalf("failed to parse config file: %v", err)
	}

	d, err := deployer.NewDeployer(cfg)
	if err != nil {
		log.Fatalf("failed to create deployer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Minute)
	defer cancel()

	groupsNodes := map[string][]int{}

	pass := []string{}
	fail := map[string]error{}

	for _, group := range cfg.NodeGroups {
		nodes, err := d.FilterNodes(group, ctx)
		if err != nil {
			fail[group.Name] = err
			continue
		}

		nodesIDs := []int{}
		for _, node := range nodes {
			nodesIDs = append(nodesIDs, node.NodeID)
		}
		groupsNodes[group.Name] = nodesIDs
	}

	vmsWorkloads, disksWorkloads := d.ParseVms(cfg.Vms, groupsNodes, cfg.SSHKey)
	var lock sync.Mutex
	var wg sync.WaitGroup

	deploymentStart := time.Now()

	for group, vms := range vmsWorkloads {
		wg.Add(1)
		go func(group string, vms []workloads.VM) {
			defer wg.Done()
			err := d.MassDeploy(ctx, vms, groupsNodes[group], disksWorkloads[group])

			lock.Lock()
			defer lock.Unlock()

			if err != nil {
				fail[group] = err
			} else {
				pass = append(pass, group)
			}
		}(group, vms)
	}
	wg.Wait()

	fmt.Println("deployment took ", time.Since(deploymentStart))
	fmt.Println("ok:")
	for _, group := range pass {
		fmt.Printf("\t%s\n", group)
	}

	fmt.Println("error:")
	for group, err := range fail {
		fmt.Printf("\t%s: %v\n", group, err)
	}
}
