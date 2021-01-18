package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

const version = "0.1"

func handleError(err error) {
	log.Fatalln(err)
}

var app = cli.NewApp()

func main() {
	app := &cli.App{
		Name:      "Rosti.cz CLI",
		Usage:     "CLI application to manage projects hosted on Rosti.cz",
		UsageText: "This command line tool reads Rostifile located in the current work directory and runs different commands with parameters defined in this file.\n\n     rostictl [global options] command [command options] [arguments...]",
		Flags:     []cli.Flag{},
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
