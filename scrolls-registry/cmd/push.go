package cmd

import (
	"fmt"
	"github.com/highcard-dev/scrolls-registry/internal/registry"
	"github.com/highcard-dev/scrolls-registry/internal/util/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var pushType string

type PushType string

var (
	PushTypePackages            PushType = "packages"
	PushTypeRegistryIndex       PushType = "registry-index"
	PushTypeTranslations        PushType = "translations"
	PushTypeTranslationsScroll  PushType = "translations-scroll"
	PushTypeTranslationsVariant PushType = "translations-variant"
	PushTypeRegex                        = map[PushType]string{
		PushTypePackages:            "[^w].tar.gz",
		PushTypeRegistryIndex:       ".registry{1}",
		PushTypeTranslationsScroll:  `\w+(/){1}\w+(/.meta/){1}(de-DE|en-US){1}.md{1}`,
		PushTypeTranslationsVariant: `\w+(\/){1}\w+(\/){1}.*(/.meta/){1}(de-DE|en-US){1}.md{1}`,
	}
)

var PushCommand = &cobra.Command{
	Use:   "push",
	Short: "Package scrolls and generate updated version of .registry file",
	Run: func(cmd *cobra.Command, args []string) {
		defer logger.Log.Sync()
		client, err := registry.NewS3Client(os.Getenv("SCROLL_REGISTRY_ENDPOINT"), os.Getenv("SCROLL_REGISTRY_BUCKET"), os.Getenv("SCROLL_REGISTRY_API_KEY"), os.Getenv("SCROLL_REGISTRY_API_SECRET"))
		if err != nil {
			logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextPush), zap.Error(err))
		}

		pushTypeConverted := PushType(pushType)
		switch pushTypeConverted {
		case PushTypePackages, PushTypeRegistryIndex:
			files, err := filterFiles(buildsDir, PushTypeRegex[pushTypeConverted])
			if err != nil {
				logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextPush), zap.Error(err))
			}
			if len(files) <= 0 {
				logger.Log.Warn("No files were found", zap.String("type", string(pushTypeConverted)))
				return
			}
			for _, filePath := range files {
				_, fileName := filepath.Split(filePath)
				err := client.PutObject("./"+filePath, fileName)
				if err != nil {
					logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextPush), zap.Error(err))
				}
				logger.Log.Info("File uploaded", zap.String("type", string(pushTypeConverted)), zap.String("file", fileName))
			}
			break
		case PushTypeTranslations:
			filesScroll, err := filterFiles(scrollsDir, PushTypeRegex[PushTypeTranslationsScroll])
			if err != nil {
				logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextPush), zap.Error(err))
			}
			if len(filesScroll) <= 0 {
				logger.Log.Warn("No files were found", zap.String("type", string(PushTypeTranslationsScroll)))
				return
			}
			filesVariant, err := filterFiles(scrollsDir, PushTypeRegex[PushTypeTranslationsVariant])
			if err != nil {
				logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextPush), zap.Error(err))
			}
			if len(filesVariant) <= 0 {
				logger.Log.Warn("No files were found", zap.String("type", string(PushTypeTranslationsVariant)))
				return
			}
			for _, filePath := range filesScroll {
				key := getKeyByType(PushTypeTranslationsScroll, filePath)
				err := client.PutObject("./"+filePath, key)
				if err != nil {
					logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextPush), zap.Error(err))
				}
				logger.Log.Info("File uploaded",
					zap.String("type", string(pushTypeConverted)),
					zap.String("sub-type", string(PushTypeTranslationsScroll)),
					zap.String("file", key))
			}
			for _, filePath := range filesVariant {
				key := getKeyByType(PushTypeTranslationsVariant, filePath)
				err := client.PutObject("./"+filePath, key)
				if err != nil {
					logger.Log.Fatal("fatal", zap.String(logger.LogKeyContext, logger.LogContextPush), zap.Error(err))
				}
				logger.Log.Info("File uploaded",
					zap.String("type", string(pushTypeConverted)),
					zap.String("sub-type", string(PushTypeTranslationsVariant)),
					zap.String("file", key))
			}
			break
		}
	},
}

func getKeyByType(pushType PushType, path string) string {
	fileDir, fileName := filepath.Split(filepath.Clean(path))
	d := strings.TrimPrefix(fileDir, filepath.Clean(scrollsDir))
	dirParts := strings.Split(d, "/")
	switch pushType {
	case PushTypeTranslationsScroll:
		return fmt.Sprintf(".i18n/scroll/%s/%s", dirParts[1], fileName)
	case PushTypeTranslationsVariant:
		return fmt.Sprintf(".i18n/scroll/%s/%s/%s", dirParts[1], dirParts[2], fileName)
	}
	return ""
}

func filterFiles(dirPath string, regexPattern string) ([]string, error) {

	var selectedFiles []string
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, err
	}
	err = filepath.Walk(dirPath,
		func(path string, info os.FileInfo, err error) error {
			if regex.MatchString(filepath.Clean(path)) && !strings.Contains(path, ".sample") {
				selectedFiles = append(selectedFiles, path)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return selectedFiles, nil
}

func init() {
	PushCommand.Flags().StringVarP(&pushType, "push-type", "t", "packages", "What should be pushed to the registry")
	PushCommand.Flags().StringVarP(&buildsDir, "builds-dir", "b", "./.builds", "Directory in which the built scrolls should be placed")

}
