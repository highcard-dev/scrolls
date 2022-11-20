package cmd

import (
	"github.com/highcard-dev/scrolls-registry/internal/util/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
)

var CleanCommand = &cobra.Command{
	Use:   "clean",
	Short: "Clean build folder",
	Run: func(cmd *cobra.Command, args []string) {
		err := os.RemoveAll(buildsDir)
		if err != nil {
			logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextPush), zap.Error(err))
		}
	},
}

func init() {
	CleanCommand.Flags().StringVarP(&buildsDir, "builds-dir", "b", "./.builds", "Directory in which the built scrolls should be placed")
}
