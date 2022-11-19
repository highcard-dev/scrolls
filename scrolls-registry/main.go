package main

import (
	"github.com/highcard-dev/scrolls-registry/cmd"
	"github.com/highcard-dev/scrolls-registry/internal/util/logger"
	"go.uber.org/zap"
)

func main() {
	defer logger.Log.Sync()
	cmd.RootCmd.AddCommand(cmd.UpdateCommand)
	cmd.RootCmd.AddCommand(cmd.PushCommand)
	cmd.RootCmd.AddCommand(cmd.PrintCommand)
	err := cmd.RootCmd.Execute()
	if err != nil {
		logger.Log.Fatal("fatal error on command", zap.Error(err))
	}
}
