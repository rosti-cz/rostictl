package main

import (
	"log"
	"os"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

const version = "0.8"

func handleError(err error) {
	log.Fatalln(err)
}

var app = cli.NewApp()

var cRed *color.Color = color.New(color.FgRed)
var cYellow *color.Color = color.New(color.FgYellow)
var cGreen *color.Color = color.New(color.FgGreen)
var cWhite *color.Color = color.New(color.FgWhite)
var cGrey *color.Color = color.New(color.FgWhite)

func main() {
	app := &cli.App{
		Name:      "Rosti.cz CLI",
		Usage:     "CLI application to manage projects hosted on Rosti.cz",
		UsageText: "This command line tool reads Rostifile located in the current work directory and runs different commands with parameters defined in this file.\n\n     rostictl [global options] command [command options] [arguments...]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "no-color",
				Aliases: []string{"nocolor"},
				Usage:   "Terminal output without colors",
			},
		},
		Before: noColor,
		Commands: []*cli.Command{
			{
				Name:    "up",
				Aliases: []string{},
				Usage:   "Deploys new or existing application",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "company",
						Aliases: []string{"c"},
						Value:   0,
						Usage:   "Company ID",
					},
					&cli.BoolFlag{
						Name:  "force-init",
						Usage: "Runs initialization commands even if the application exists.",
					},
				},
				Action: commandUp,
			},
			{
				Name:    "down",
				Aliases: []string{"stop"},
				Usage:   "Turns the application off but doesn't remove it",
				Action:  commandDown,
			},
			{
				Name:    "start",
				Aliases: []string{},
				Usage:   "Turns the application on without deploying any code",
				Action:  commandStart,
			},
			{
				Name:    "remove",
				Aliases: []string{"rm"},
				Usage:   "Removes the application",
				Action:  commandRemove,
			},
			{
				Name:    "status",
				Aliases: []string{},
				Usage:   "Returns status of the application",
				Action:  commandStatus,
			},
			{
				Name:    "plans",
				Aliases: []string{},
				Usage:   "Prints list of available plans you can use in Rostifile",
				Action:  commandPlans,
			},
			{
				Name:    "companies",
				Aliases: []string{},
				Usage:   "Prints list of companies you are member of.",
				Action:  commandCompanies,
			},
			{
				Name:    "runtimes",
				Aliases: []string{},
				Usage:   "Prints list of available runtimes you can use in Rostifile",
				Action:  commandRuntimes,
			},
			{
				Name:    "init",
				Aliases: []string{},
				Usage:   "Creates a new Rostifile in the current working directory",
				Action:  commandInit,
			},
			{
				Name:    "rostifile",
				Aliases: []string{},
				Usage:   "Read Rostifile into internal structure and encodes it again (for debugging)",
				Action:  commandRostifile,
			},
			{

				Name:    "import",
				Aliases: []string{},
				Usage:   "Imports existing application into a current working directory (creates .rosti.state, rewrites existing one if there is any)",
				Action:  commandImport,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "company",
						Aliases: []string{"c"},
						Value:   0,
						Usage:   "Company ID",
					},
				},
			},
			{
				Name:    "version",
				Aliases: []string{},
				Usage:   "Prints version of this binary",
				Action:  commandVersion,
			},
			// backup
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
