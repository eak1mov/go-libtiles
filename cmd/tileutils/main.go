package main

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(&convertCmd{}, "")
	subcommands.Register(&exportCmd{}, "")
	subcommands.Register(&importCmd{}, "")

	flag.Parse()
	os.Exit(int(subcommands.Execute(context.Background())))
}
