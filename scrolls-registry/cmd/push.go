package cmd

import (
	"github.com/highcard-dev/scrolls-registry/internal/registry"
	"github.com/highcard-dev/scrolls-registry/internal/util/logger"
	"github.com/spf13/cobra"
	"os"
)

var pushType string

var PushCommand = &cobra.Command{
	Use:   "push",
	Short: "Package scrolls and generate updated version of .registry file",
	Run: func(cmd *cobra.Command, args []string) {
		defer logger.Log.Sync()
		registry.NewClient(os.Getenv("SCROLL_REGISTRY_ENDPOINT"), os.Getenv("SCROLL_REGISTRY_API_KEY"), os.Getenv("SCROLL_REGISTRY_API_SECRET"))
		switch pushType {
		case "packages":

			break
		case "registry-index":
			break
		case "translations":
			break
		}
	},
}

func init() {
	PushCommand.Flags().StringVarP(&pushType, "push-type", "t", "packages", "What should be pushed to the registry")
}
