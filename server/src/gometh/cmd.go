package gometh

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// C is the package config
var C Config

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "gometh",
	Short: "A child chain for geth",
	Long:  "A child chain for geth",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var runCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server",
	Long:  "Start the server",
	Run: func(cmd *cobra.Command, args []string) {
		json, _ := json.MarshalIndent(C, "", "  ")
		log.Println("Efective configuration: " + string(json))
		startServer()
	},
}

// ExecuteCmd adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func ExecuteCmd() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)

	}
}

var patata string

func init() {

	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gometh.yaml)")
	RootCmd.AddCommand(runCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	viper.SetConfigType("yaml")
	viper.SetConfigName("gometh") // name of config file (without extension)
	viper.AddConfigPath("$HOME")  // adding home directory as first search path
	viper.SetEnvPrefix("GOMETH")  // so viper.AutomaticEnv will get matching envvars starting with O2M_
	viper.AutomaticEnv()          // read in environment variables that match

	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	// If a config file is found, read it in.

	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", cfgFile)
		if err := viper.Unmarshal(&C); err != nil {
			panic(err)
		}
	} else {
		log.Fatalln("Configuration file ~/.gometh.yaml not found.")
	}

	if C.DataPath == "" {
		usr, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		C.DataPath = usr.HomeDir + "/.gometh"
	}

}
