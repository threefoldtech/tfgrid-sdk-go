package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cosmos/go-bip39"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/stellar/go/support/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/server"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/version"
)

// farmerBotCmd represents the root base command when called without any subcommands
var farmerBotCmd = &cobra.Command{
	Use:   "farmerbot",
	Short: "Run farmerbot to manage your farms",
	Long:  fmt.Sprintf(`Welcome to the farmerbot (%v). The farmerbot is a service that a farmer can run allowing him to automatically manage the nodes of his farm.`, version.Version),
	RunE: func(cmd *cobra.Command, args []string) error {
		network, mnemonic, redisAddr, err := getDefaultFlags(cmd)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to parse flags")
		}

		config, err := cmd.Flags().GetString("config")
		if err != nil {
			log.Fatal().Err(err).Msgf("invalid config dir path input '%s'", config)
		}

		farmerBot, err := server.NewFarmerBot(cmd.Context(), config, network, mnemonic, redisAddr)
		if err != nil {
			log.Fatal().Err(err).Msg("farmerbot failed to start")
		}

		go farmerBot.Run(cmd.Context())
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	farmerBotCmd.AddCommand(versionCmd)

	err := farmerBotCmd.Execute()
	if err != nil {
		log.Fatal().Err(err).Send()
		os.Exit(1)
	}
}

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	farmerBotCmd.Flags().StringP("config", "c", "", "enter your config file that includes your farm, node and power configs")
	farmerBotCmd.Flags().StringP("redis", "r", "", "the address of the redis db")

	farmerBotCmd.PersistentFlags().StringP("network", "n", "dev", "the grid network to use")
	farmerBotCmd.PersistentFlags().StringP("mnemonic", "m", "", "the mnemonic of the account of the farmer")
	farmerBotCmd.PersistentFlags().BoolP("debug", "d", false, "by setting this flag the farmerbot will print debug logs too")
	farmerBotCmd.PersistentFlags().StringP("log", "l", "farmerbot.log", "enter your log file path to debug")
}

func getDefaultFlags(cmd *cobra.Command) (network string, mnemonic string, redisAddr string, err error) {
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		err = errors.Wrapf(err, "invalid log debug mode input '%v'", debug)
		return
	}

	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	logPath, err := cmd.Flags().GetString("log")
	if err != nil {
		err = errors.Wrapf(err, "invalid log file path input '%s'", logPath)
		return
	}

	logFile, err := os.OpenFile(
		logPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0664,
	)
	if err != nil {
		err = errors.Wrapf(err, "failed to open file '%s'", logPath)
		return
	}

	multiWriter := zerolog.MultiLevelWriter(os.Stdout, logFile)
	log.Logger = zerolog.New(multiWriter).With().Timestamp().Logger().Output(zerolog.ConsoleWriter{Out: os.Stderr})

	redisAddr, err = cmd.Flags().GetString("redis")
	if err != nil {
		err = errors.Wrapf(err, "invalid redis address input '%s'", redisAddr)
		return
	}

	if len(strings.TrimSpace(redisAddr)) == 0 {
		err = errors.New("redis address is required")
		return
	}

	network, err = cmd.Flags().GetString("network")
	if err != nil {
		err = errors.Wrapf(err, "invalid network input '%s'", network)
		return
	}

	if network != "dev" && network != "qa" && network != "test" && network != "main" {
		err = fmt.Errorf("network must be one of dev, qa, test, and main not '%s'", network)
		return
	}

	mnemonic, err = cmd.Flags().GetString("mnemonic")
	if err != nil {
		err = errors.Wrapf(err, "invalid mnemonic input '%s'", mnemonic)
		return
	}

	if len(strings.TrimSpace(mnemonic)) == 0 {
		err = errors.New("mnemonic is required")
		return
	}

	if !bip39.IsMnemonicValid(mnemonic) {
		err = fmt.Errorf("mnemonic '%s' is invalid", mnemonic)
	}

	return
}
