package cmd

import (
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var config Config

// Config is struct of config
type Config struct {
	Template []string
	Git GitConfig
}

// GitConfig is struct of config related to git
type GitConfig struct {
	Heading string
	Repositories []string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nippo",
	Short: "Generate nippo",
}

// Execute command
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.nippo.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".nippo" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".nippo")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match
}
