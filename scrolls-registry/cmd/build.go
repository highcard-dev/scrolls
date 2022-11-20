package cmd

import (
	"fmt"
	"github.com/highcard-dev/scrolls-registry/internal/registry"
	"github.com/highcard-dev/scrolls-registry/internal/util"
	"github.com/highcard-dev/scrolls-registry/internal/util/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
)

var onlyChanged bool

var UpdateCommand = &cobra.Command{
	Use:   "build",
	Short: "Package scrolls and generate updated version of .registry file",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := registry.NewClient(os.Getenv("SCROLL_REGISTRY_ENDPOINT"), os.Getenv("SCROLL_REGISTRY_BUCKET"), os.Getenv("SCROLL_REGISTRY_API_KEY"), os.Getenv("SCROLL_REGISTRY_API_SECRET"))
		if err != nil {
			logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
		}
		currentRegistry, err := client.GetRegistry()
		if err != nil {
			logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
		}
		scrolls, err := os.ReadDir(scrollsDir)
		if err != nil {
			logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
		}
		err = util.RecreateDir(buildsDir)
		if err != nil {
			logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
		}

		for _, scroll := range scrolls {
			if scroll.Name()[:1] == "." {
				//.sample
				continue
			}
			scrollPath := fmt.Sprintf("%s/%s", scrollsDir, scroll.Name())
			variants, err := os.ReadDir(scrollPath)
			if err != nil {
				logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
			}
			for _, variant := range variants {
				if variant.Name()[:1] == "." {
					//.meta
					continue
				}
				variantPath := fmt.Sprintf("%s/%s", scrollPath, variant.Name())
				versions, err := os.ReadDir(variantPath)
				if err != nil {
					logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
				}

				for _, version := range versions {
					if version.Name()[:1] == "." {
						//.meta
						continue
					}
					versionPath := fmt.Sprintf("%s/%s", variantPath, version.Name())
					scrollFile, err := registry.OpenScrollYaml(fmt.Sprintf("%s/scroll.yaml", versionPath))
					if err != nil {
						logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
					}
					registryEntry, ok := currentRegistry[registry.Variant(variant.Name())][registry.VariantVersion(version.Name())]
					if ok {
						//already exists in the registry
						if !onlyChanged || (onlyChanged && !scrollFile.Version.Equal(registryEntry.Latest.Version)) {
							entry, shouldUpdateLatest := registryEntry.Refresh(scrollFile.Version)
							currentRegistry[registry.Variant(variant.Name())][registry.VariantVersion(version.Name())] = entry
							if err := generatePackage(versionPath, variant.Name(), version.Name(), scrollFile.Version); err != nil {
								logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
							}
							if shouldUpdateLatest {
								if err := generatePackage(versionPath, variant.Name(), version.Name(), "latest"); err != nil {
									logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
								}
							}
						}
					} else {
						//new to the registry
						if err := generatePackage(versionPath, variant.Name(), version.Name(), scrollFile.Version); err != nil {
							logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
						}
						if err := generatePackage(versionPath, variant.Name(), version.Name(), "latest"); err != nil {
							logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
						}
						currentRegistry[registry.Variant(variant.Name())][registry.VariantVersion(version.Name())] = registry.NewRegistryEntry(scrollFile.Version)
					}
				}
			}
		}
		err = util.CreateYamlFile(currentRegistry, fmt.Sprintf("%s/.registry", buildsDir))
		if err != nil {
			logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextUpdate), zap.Error(err))
		}
	},
}

func generatePackage[T registry.SemanticVersion | string](srcPath string, variant string, version string, scrollVersion T) error {
	var destinationPath string
	if version == "latest" {
		destinationPath = fmt.Sprintf("%s/%s:%s.tar.gz", buildsDir, variant, scrollVersion)
	} else {
		destinationPath = fmt.Sprintf("%s/%s@%s:%s.tar.gz", buildsDir, variant, version, scrollVersion)
	}
	logger.Log.Info("generating package...", zap.String("package", destinationPath))
	if err := util.TarDirectory(srcPath, destinationPath); err != nil {
		return err
	}
	logger.Log.Info("generated package", zap.String("package", destinationPath))
	return nil
}

func init() {
	UpdateCommand.Flags().StringVarP(&buildsDir, "builds-dir", "b", "./.builds", "Directory in which the built scrolls should be placed")
	UpdateCommand.Flags().BoolVarP(&onlyChanged, "only-changed", "c", false, "Should rebuild only changed scrolls")
}
