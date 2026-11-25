package main

import (
	"os"

	"acovia.net/record"
	"github.com/fovlin/opt"
)

var (
	commands = map[string]func() error{
		"scan":  scanCmd,
		"ping":  pingCmd,
		"help":  help,
		"about": about,
	}

	cmd        string
	subCmdArgs []string
)

func main() {
	if err := initProgram(); err != nil {
		record.Error(err)
	}

	function, ok := commands[cmd]
	if !ok {
		record.Error("unknown command:", cmd)
	}
	if err := function(); err != nil {
		record.Error(err)
	}
}

func initProgram() error {
	opt.StringVar(&timeoutFlag, "t", "timeout", "")
	opt.StringVar(&workersFlag, "w", "workers", "256")
	opt.StringVar(&portsFlag, "p", "ports", "1-65535")
	opt.StringVar(&rateFlag, "r", "rate", "128")
	opt.BoolSwitch(&onlyMC, "o", "only-mc")
	opt.BoolFunc("", "help", func() {
		help()
		os.Exit(0)
	})

	if err := opt.Parse(); err != nil {
		return err
	}

	args := opt.Args()
	if len(args) < 1 {
		args = []string{"help"}
	}
	cmd = args[0]
	subCmdArgs = args[1:]
	return nil
}
