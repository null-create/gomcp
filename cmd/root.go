package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgfile string

	rootCmd = &cobra.Command{
		Use:   "gomcp",
		Short: "Root command for gomcp",
	}
)

func init() {
	initConfig()
}

func initConfig() {
	if cfgfile != "" {
		viper.SetConfigFile(cfgfile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cobra")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Fatal(fmt.Sprintf("Using config file: %v", viper.ConfigFileUsed()))
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
