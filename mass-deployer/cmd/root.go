package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/internal/parser"
	deployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
)

var rootCmd = &cobra.Command{
	Use:   "mass-deployer",
	Short: "A tool for deploying groups of vms on Threefold Grid",

	Run: func(cmd *cobra.Command, args []string) {
		configFile, err := cmd.Flags().GetString("config")
		if err != nil || configFile == "" {
			log.Error().Err(err).Msg("error in config file")
			return
		}
		err = runDeployer(configFile)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to parse config file")
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringP("config", "c", "", "path to config file")
}

func runDeployer(configFile string) error {
	cfg, err := parser.ParseConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	d, err := deployer.NewDeployer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create deployer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Minute)
	defer cancel()

	groupsNodes := map[string][]int{}

	pass := map[string][]deployer.VMInfo{}
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

	vmsWorkloads, disksWorkloads := d.ParseVms(cfg.Vms, groupsNodes, cfg.SSHKeys)
	var lock sync.Mutex
	var wg sync.WaitGroup

	deploymentStart := time.Now()

	for group, vms := range vmsWorkloads {
		wg.Add(1)
		go func(group string, vms []workloads.VM) {
			defer wg.Done()
			info, err := d.MassDeploy(ctx, vms, groupsNodes[group], disksWorkloads[group])

			lock.Lock()
			defer lock.Unlock()

			if err != nil {
				fail[group] = err
			} else {
				pass[group] = info
			}
		}(group, vms)
	}
	wg.Wait()

	fmt.Println("deployment took ", time.Since(deploymentStart))
	fmt.Println("ok:")
	for group, info := range pass {

		groupInfo, err := yaml.Marshal(info)
		if err != nil {
			log.Debug().Err(err).Msg("failed to marshal json")
		}
		fmt.Printf("%s: \n%v\n", group, string(groupInfo))
	}

	fmt.Println("error:")
	for group, err := range fail {
		fmt.Printf("%s: %v\n", group, err)
	}
	return nil
}
