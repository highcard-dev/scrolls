package cmd

import (
	"fmt"
	"github.com/highcard-dev/scrolls-registry/internal/registry"
	"github.com/highcard-dev/scrolls-registry/internal/util/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
)

var PrintCommand = &cobra.Command{
	Use:   "print",
	Short: "Print remote .registry file",
	RunE: func(cmd *cobra.Command, args []string) error {
		defer logger.Log.Sync()
		client, err := registry.NewS3Client(os.Getenv("SCROLL_REGISTRY_ENDPOINT"), os.Getenv("SCROLL_REGISTRY_BUCKET"), os.Getenv("SCROLL_REGISTRY_API_KEY"), os.Getenv("SCROLL_REGISTRY_API_SECRET"))
		if err != nil {
			logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextView), zap.Error(err))
		}
		registry, err := client.GetRegistry()
		if err != nil {
			logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
		}
		for variantName, variant := range registry {
			for versionName, version := range variant {
				logger.Log.Info(fmt.Sprintf("------------ Scroll: %s@%s ------------", variantName, versionName))
				logger.Log.Info(fmt.Sprintf("Latest: %s", version.Latest.String()))
			}
		}
		return nil
	},
}