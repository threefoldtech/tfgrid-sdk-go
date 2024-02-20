package deployer

import (
	"fmt"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
)

func RunCanceler(cfg Config, tfPluginClient deployer.TFPluginClient, debug bool) error {
	for _, group := range cfg.NodeGroups {
		// try to delete group with the new name format "vm/<group name>"
		name := fmt.Sprintf("vm/%s", group.Name)
		err := tfPluginClient.CancelByProjectName(name, true)
		if err != nil {
			fmt.Println("here??")
			// if cancelation failed try to delete group with the old name format "<group name>"
			err := tfPluginClient.CancelByProjectName(group.Name, true)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
