package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define a global flag (optional)
	verbose := flag.Bool("v", false, "enable verbose output")

	// Parse global flags
	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Println("expected 'build' or 'deploy' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		fooCmd := flag.NewFlagSet("build", flag.ExitOnError)
		fooEnable := fooCmd.Bool("enable", false, "enable foo functionality")
		fooName := fooCmd.String("name", "", "name for foo")
		fooCmd.Parse(os.Args[2:]) // Parse subcommand-specific flags

		fmt.Printf("Foo command executed: enable=%t, name=%s, verbose=%t\n", *fooEnable, *fooName, *verbose)
		build()

	case "deploy":
		barCmd := flag.NewFlagSet("deploy", flag.ExitOnError)
		barLevel := barCmd.Int("level", 0, "level for bar")
		barCmd.Parse(os.Args[2:]) // Parse subcommand-specific flags

		fmt.Printf("Bar command executed: level=%d, verbose=%t\n", *barLevel, *verbose)
		deploy()

	default:
		fmt.Println("unknown subcommand:", os.Args[1])
		os.Exit(1)
	}
}
