package deployer

import "github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"

func RunCanceler(cfg Config, tfPluginClient deployer.TFPluginClient, debug bool) error {
	for _, group := range cfg.NodeGroups {
		err := tfPluginClient.CancelByProjectName(group.Name, true)
		if err != nil {
			return err
		}
	}
	return nil
}
