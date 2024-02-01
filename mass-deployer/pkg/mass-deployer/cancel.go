package deployer

func RunCanceler(cfg Config, debug bool) error {
	tfPluginClient, err := setup(cfg, debug)
	if err != nil {
		return err
	}

	for _, group := range cfg.NodeGroups {
		err = tfPluginClient.CancelByProjectName(group.Name)
		if err != nil {
			return err
		}
	}
	return nil
}
