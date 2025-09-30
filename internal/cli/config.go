package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read/write global reqo configuration (stored in ~/.reqo/config.yaml)",
	}
	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Args:  cobra.ExactArgs(2),
		Short: "Set a global config value",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, val := args[0], args[1]
			viper.Set(key, val)
			if err := viper.WriteConfig(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s = %s\n", key, val)
			return nil
		},
	}
	cmd.AddCommand(setCmd)

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Args:  cobra.ExactArgs(1),
		Short: "Show a config value",
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			val := viper.Get(key)
			fmt.Fprintf(cmd.OutOrStdout(), "%s = %v\n", key, val)
			return nil
		},
	}
	cmd.AddCommand(getCmd)

	// init viper on first use
	cobra.OnInitialize(func() {
		home, _ := os.UserHomeDir()
		viper.SetConfigFile(home + "/.reqo/config.yaml")
		_ = viper.ReadInConfig() // ignore error – file may not exist yet
	})
	return cmd
}
