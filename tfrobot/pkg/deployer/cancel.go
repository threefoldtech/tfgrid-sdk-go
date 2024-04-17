package deployer

import (
	"fmt"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
)

func RunCanceler(cfg Config, tfPluginClient deployer.TFPluginClient, debug bool) error {
	for _, group := range cfg.NodeGroups {

		// try to delete group twice with the new and old name formats
		// tfrobot shouldn't create two projects with both new and old name format
		// so only one of the two `CancelByProjectName` would have effect

		name := fmt.Sprintf("vm/%s", group.Name)
		err := tfPluginClient.CancelByProjectName(name, true)
		if err != nil {
			return err
		}

		err = tfPluginClient.CancelByProjectName(group.Name, true)
		if err != nil {
			return err
		}
	}
	return nil
}
