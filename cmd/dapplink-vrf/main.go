package main

import (
	"context"
	"os"

	"github.com/ethereum/go-ethereum/log"

	"github.com/the-web3/dapplink-vrf/common/opio"
)

var (
	GitCommit = ""
	GitData   = ""
)

func main() {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))
	app := NewCli(GitCommit, GitData)
	ctx := opio.WithInterruptBlocker(context.Background())
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Error("Application failed", err)
		os.Exit(1)
	}
}
