package deployer

func RunCanceler(cfg Config) error {
	tfPluginClient, err := setup(cfg)
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
