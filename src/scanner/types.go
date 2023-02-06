package scanner

import "github.com/rosti-cz/cli/src/parser"

// RostifileBits contains bits of Rostifile that are recommended to setup in newly created Rostifile.
// It is used in init command.
type RostifileBits struct {
	Technology     string
	Processes      []parser.Process
	AppPort        int
	BeforeCommands []string
	AfterCommands  []string
}

// PackageJSON is used to test if scripts field contains start script
type PackageJSON struct {
	Scripts map[string]string `json:"scripts"`
}
