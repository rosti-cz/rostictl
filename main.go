package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func handleError(err error) {
	log.Fatalln(err)
}

var app = cli.NewApp()

func main() {
	app := &cli.App{
		Name:      "Rosti.cz CLI",
		Usage:     "CLI application to manage projects hosted on Rosti.cz",
		UsageText: "This command line tool reads Rostifile located in the current work directory and runs different commands with parameters defined in this file.\n\n     rostictl [global options] command [command options] [arguments...]",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "company",
				Aliases: []string{"c"},
				Value:   0,
				Usage:   "Company ID",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "up",
				Aliases: []string{},
				Usage:   "Deploys new or existing application.",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "company",
						Aliases: []string{"c"},
						Value:   0,
						Usage:   "Company ID",
					},
				},
				Action: commandUP,
			},
			{
				Name:    "down",
				Aliases: []string{},
				Usage:   "Turns the application off but doesn't remove it.",
				Action: func(c *cli.Context) error {
					fmt.Println("")

					return nil
				},
			},
			{
				Name:    "rm",
				Aliases: []string{},
				Usage:   "Removes the application.",
				Action: func(c *cli.Context) error {
					fmt.Println("")

					return nil
				},
			},
			{
				Name:    "status",
				Aliases: []string{},
				Usage:   "Returns status of the application.",
				Action: func(c *cli.Context) error {
					fmt.Println("")

					return nil
				},
			},
			// plans
			// companies
			// runtimes
			// backup
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
