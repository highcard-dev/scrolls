package cmd

import (
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var scrollsDir, envPath, buildsDir string
var RootCmd = &cobra.Command{
	Use:   "scrolls-registry",
	Short: "Druid Scroll Registry Tool",
	Long:  `An application that enable managing scrolls versioning and packaging`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		godotenv.Load(envPath)
	},
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&scrollsDir, "scrolls-dir", "d", "./", "Directory in which the scrolls files are placed")
	RootCmd.PersistentFlags().StringVarP(&envPath, "env-file", "e", "./../.env", "Path to environment file (.env)")
}
