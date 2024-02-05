package main

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
	"github.com/mkrcx/mkrcx/src/leash/commands"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&commands.LaunchCmd{}, "")
	subcommands.Register(&commands.NewUserCmd{}, "")
	subcommands.Register(&commands.NewServiceUserCmd{}, "")
	subcommands.Register(&commands.NewApiKeyCmd{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
