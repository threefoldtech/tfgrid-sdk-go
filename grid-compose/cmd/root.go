package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/pkg/types"
	"gopkg.in/yaml.v2"
)

var (
	app        App
	configFile string
	network    string
	mnemonic   string
)

var rootCmd = &cobra.Command{
	Use:   "grid-compose",
	Short: "Grid-Compose is a tool for running multi-vm applications on TFGrid defined using a Yaml formatted file.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var err error
		app, err = NewApp(network, mnemonic, configFile)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
	},
}

func init() {
	network = os.Getenv("NETWORK")
	mnemonic = os.Getenv("MNEMONIC")
	rootCmd.PersistentFlags().StringVarP(&configFile, "file", "f", "./grid-compose.yaml", "the grid-compose configuration file")

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
}

type App struct {
	Client deployer.TFPluginClient
	Specs  types.Specs
}

func NewApp(net, mne, filePath string) (App, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return App{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return App{}, fmt.Errorf("failed to read file: %w", err)
	}

	var specs types.Specs
	if err := yaml.Unmarshal(content, &specs); err != nil {
		return App{}, fmt.Errorf("failed to parse file: %w", err)
	}

	client, err := deployer.NewTFPluginClient(mne, deployer.WithNetwork(net))
	if err != nil {
		return App{}, fmt.Errorf("failed to load grid client: %w", err)
	}

	return App{
		Specs:  specs,
		Client: client,
	}, nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Send()
	}
}
