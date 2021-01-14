package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rosti-cz/cli/src/config"
	"github.com/rosti-cz/cli/src/parser"
	"github.com/rosti-cz/cli/src/rostiapi"
	"github.com/rosti-cz/cli/src/ssh"
	"github.com/rosti-cz/cli/src/state"
	"github.com/urfave/cli/v2"
)

// Deploys or re-deploys an application
func commandUP(c *cli.Context) error {
	config := config.Load()

	fmt.Println(".. loading Rostifile")
	rostifile, err := parser.Parse()
	if err != nil {
		return err
	}

	client := rostiapi.Client{
		Token: config.Token,
	}

	fmt.Println(".. loading state file")
	appState, err := state.Load()
	if err != nil {
		return err
	}
	defer state.Write(appState)

	// Pick the right company
	fmt.Println(".. loading list of your companies")
	companies, err := client.GetCompanies()
	if err != nil {
		return err
	}

	if len(companies) == 0 {
		return errors.New("no company found")
	}

	companyIDFromFlag := uint(c.Int("company"))
	companyID := appState.CompanyID

	if companyIDFromFlag != 0 {
		companyID = companyIDFromFlag
	} else if companyID == 0 {
		if len(companies) == 1 {
			companyID = companies[0].ID
		} else if len(companies) > 1 {
			fmt.Println("You have access to multiple companies, pick one of the companies below and use -c COMPANY_ID flag to call this command.")
			fmt.Println("")
			fmt.Printf("  %6s  Company name\n", "ID")
			fmt.Printf("  %6s  ------------\n", "------")
			for _, company := range companies {
				fmt.Printf("  %6s  %s\n", strconv.Itoa(int(company.ID)), company.Name)
			}
			return nil
		} else {
			return errors.New("no company found")
		}
	}

	var found bool
	for _, company := range companies {
		if company.ID == companyID {
			found = true
			break
		}
	}
	if !found {
		return errors.New("selected company (" + strconv.Itoa(int(companyIDFromFlag)) + ") not found")
	}

	appState.CompanyID = companyID
	client.CompanyID = companyID

	// Select plan
	if rostifile.Plan == "" {
		// TODO: implements something like default plan loaded from the API (needs support in the API)
		rostifile.Plan = "start"
	}

	fmt.Println(".. loading list of available plans")
	plans, err := client.GetPlans()
	if err != nil {
		return err
	}

	var planID uint
	for _, plan := range plans {
		if strings.ToLower(plan.Name) == strings.ToLower(rostifile.Plan) {
			planID = plan.ID
		}
	}

	// Select the right runtime
	fmt.Println(".. loading list of available runtimes")
	runtimes, err := client.GetRuntimes()
	if err != nil {
		return err
	}

	var selectedRuntime string
	var lastRuntime string

	if len(runtimes) == 0 {
		return errors.New("no runtime available")
	}

	for _, runtime := range runtimes {
		if runtime.Image == rostifile.Runtime {
			selectedRuntime = rostifile.Runtime
			break
		}
		lastRuntime = runtime.Image
	}

	if selectedRuntime == "" {
		selectedRuntime = lastRuntime
	}

	// Figure out mode
	var mode string
	if rostifile.HTTPS {
		mode = "https+le"
	} else {
		mode = "http"
	}

	// Find SSH keys
	// TODO: yes, here

	// Create or update the application
	var newApp *rostiapi.App
	if appState.ApplicationID > 0 {
		// Check existence of the app
		apps, err := client.GetApps()
		if err != nil {
			return err
		}
		var found bool
		for _, app := range apps {
			if app.ID == appState.ApplicationID {
				found = true
				break
			}
		}
		if !found {
			return errors.New("application " + rostifile.Name + " not found in your account under selected company")
		}

		// Use update
		fmt.Printf(".. updating existing application %s_%d \n", rostifile.Name, appState.ApplicationID)

		app := rostiapi.App{
			ID:     appState.ApplicationID,
			Name:   rostifile.Name,
			Image:  selectedRuntime,
			Domain: rostifile.Domains,
			Mode:   mode,
			Plan:   planID,
		}

		// TODO: save assigned domains into Rostifile

		newApp, err = client.UpdateApp(&app)
		if err != nil {
			return err
		}
	} else {
		// Create a new app
		fmt.Printf(".. creating a new application %s \n", rostifile.Name)

		app := rostiapi.App{
			Name:   rostifile.Name,
			Image:  selectedRuntime,
			Domain: rostifile.Domains,
			Mode:   mode,
			Plan:   planID,
		}

		newApp, err = client.CreateApp(&app)
		if err != nil {
			return err
		}
		appState.ApplicationID = newApp.ID
	}

	// Deploy files
	fmt.Println(".. creating an archive")
	err = createArchive(rostifile.Source, "/tmp/_archive.tar") // TODO: create a proper temporary file here
	if err != nil {
		return err
	}

	if len(newApp.SSHAccess) == 0 {
		return errors.New("no SSH access found")
	}

	sshClient := ssh.Client{
		Server:     newApp.SSHAccess[0].Hostname,
		Port:       int(newApp.SSHAccess[0].Port),
		Username:   newApp.SSHAccess[0].Username,
		SSHKeyPath: "/home/cx/.ssh/id_rsa", // TODO: we need code to find or generate this
	}

	fmt.Println(".. copying the archive into the container")
	archive, err := os.Open("/tmp/_archive.tar")
	if err != nil {
		return err
	}
	defer archive.Close()

	err = sshClient.StreamFile("/srv/_archive.tar", archive)
	if err != nil {
		return err
	}

	// TODO: mkdir -p /srv/app
	// TODO: before commands
	// TODO: after commands
	// TODO: rm -rf /srv/app/* or something like that
	// TODO: mv _archive.tar /srv/app/ && cd /srv/app && tar xf _archive.tar
	// TODO: maybe let use to set a strategy?
	// TODO: let user to exclude files and directory

	fmt.Println(".. unarchiving the code in the container")
	_, err = sshClient.Run("/bin/sh -c \"mkdir -p /srv/app && mv _archive.tar /srv/app/ && cd /srv/app && tar xf _archive.tar\"")
	if err != nil {
		return err
	}

	// Aftedeploy commands

	fmt.Println("All done!")
	// TODO: print status of the application that tells user details about the app

	return nil
}
