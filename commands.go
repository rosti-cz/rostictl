package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
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
func commandUp(c *cli.Context) error {
	config := config.Load()

	// SSH key
	user, err := user.Current()
	if err != nil {
		return fmt.Errorf("getting user info error: %w", err)
	}

	privateSSHKeyPath, _, err := findSSHKey()
	if err != nil {
		fmt.Println("Your SSH key cannot be located.")
		fmt.Println("Please put your public and private RSA key (only type supported) into:")
		fmt.Println(" ", path.Join(user.HomeDir, ".ssh", "id_rsa"))
		fmt.Println(" ", path.Join(user.HomeDir, ".ssh", "id_rsa.pub"))
		fmt.Println("and try again. You generate a new key with these commands:")
		fmt.Println("")
		fmt.Println("  mkdir -p ~/.ssh")
		fmt.Println("  ssh-keygen -t rsa -f ~/.ssh/id_rsa")
		fmt.Println("")
		return fmt.Errorf("SSH key problem: %w", err)
	}

	// Rostifile and statefile
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
	companyID, err := findCompany(&client, appState, c)
	if err != nil {
		return err
	}

	appState.CompanyID = companyID
	client.CompanyID = companyID

	// Select plan
	planID, err := selectPlan(&client, rostifile)
	if err != nil {
		return err
	}

	// Select the right runtime
	selectedRuntime, err := selectRuntime(&client, rostifile)
	if err != nil {
		return err
	}

	// Figure out mode
	var mode string
	if rostifile.HTTPS {
		mode = "https+le"
	} else {
		mode = "http"
	}

	// Create or update the application
	var newApp *rostiapi.App
	// Update existing app
	if appState.ApplicationID > 0 {
		fmt.Println(".. loading current state of the application")
		app, err := client.GetApp(appState.ApplicationID)
		if err != nil {
			return err
		}

		// If it's down let's start it
		if !app.Enabled {
			fmt.Println(".. starting the application because it was off")
			err = client.DoApp(appState.ApplicationID, "start")
			if err != nil {
				return err
			}
		}

		sshPubKey, err := readLocalSSHPubKey()
		if err != nil {
			return err
		}

		// Use update
		fmt.Printf(".. updating existing application %s_%d \n", rostifile.Name, appState.ApplicationID)

		app = rostiapi.App{
			ID:      appState.ApplicationID,
			Name:    rostifile.Name,
			Image:   selectedRuntime,
			Domains: rostifile.Domains,
			Mode:    mode,
			Plan:    planID,
			SSHKeys: []string{sshPubKey},
		}

		// TODO: save assigned domains into Rostifile

		newApp, err = client.UpdateApp(&app)
		if err != nil {
			return err
		}
	} else {
		// Create a new app
		fmt.Printf(".. creating a new application %s \n", rostifile.Name)

		sshPubKey, err := readLocalSSHPubKey()
		if err != nil {
			return err
		}

		app := rostiapi.App{
			Name:    rostifile.Name,
			Image:   selectedRuntime,
			Domains: rostifile.Domains,
			Mode:    mode,
			Plan:    planID,
			SSHKeys: []string{sshPubKey},
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
		SSHKeyPath: privateSSHKeyPath,
	}

	// TODO: Check if the SSH connect is successful and if not, wait a little bit

	fmt.Println(".. copying archive into the container")
	archive, err := os.Open("/tmp/_archive.tar")
	if err != nil {
		return err
	}
	defer archive.Close()

	err = sshClient.StreamFile("/srv/_archive.tar", archive)
	if err != nil {
		return err
	}

	var buf *bytes.Buffer

	for _, cmd := range rostifile.BeforeCommands {
		fmt.Printf(".. running command: %s\n", cmd)
		buf, err = sshClient.Run(cmd)
		if err != nil {
			fmt.Println("Command '" + cmd + "' error:")
			fmt.Println(buf.String())
			return err
		}
	}

	fmt.Println(".. unarchiving code in the container")
	cmd := "/bin/sh -c \"mkdir -p /srv/app && mv _archive.tar /srv/app/ && cd /srv/app && tar xf _archive.tar\""
	buf, err = sshClient.Run(cmd)
	if err != nil {
		fmt.Println("Unarchiving error. Command output:")
		fmt.Println(buf.String())
		return err
	}

	for _, cmd := range rostifile.AfterCommands {
		fmt.Printf(".. running command: %s\n", cmd)
		buf, err = sshClient.Run(cmd)
		if err != nil {
			fmt.Println("Command '" + cmd + "' error:")
			fmt.Println(buf.String())
			return err
		}
	}

	fmt.Println(".. all done, let's check status of the application")

	fmt.Println(".. loading current application status")
	status, err := client.GetAppStatus(appState.ApplicationID)
	if err != nil {
		return err
	}

	fmt.Println(".. loading current application configuration")
	app, err := client.GetApp(appState.ApplicationID)

	fmt.Println("")
	printAppStatus(app.Domains, status, app)

	fmt.Println("")
	fmt.Println("Note: This output doesn't have to be precise, because container")
	fmt.Println("hasn't had to boot up fully or DNS hasn't propagated into the world.")
	fmt.Println("Run `rostictl status` to run these checks again later to find out what's")
	fmt.Println("the status of this application.")

	return nil
}

func commandDown(c *cli.Context) error {
	config := config.Load()

	fmt.Println(".. loading state file")
	appState, err := state.Load()
	if err != nil {
		return err
	}
	defer state.Write(appState)

	client := rostiapi.Client{
		Token:     config.Token,
		CompanyID: appState.CompanyID,
	}

	fmt.Println(".. loading Rostifile")
	rostifile, err := parser.Parse()
	if err != nil {
		return err
	}

	fmt.Printf(".. stopping application %s_%d\n", rostifile.Name, appState.ApplicationID)
	err = client.DoApp(appState.ApplicationID, "stop")
	if err != nil {
		return err
	}

	fmt.Println(".. all done!")

	return nil
}

func commandStart(c *cli.Context) error {
	config := config.Load()

	fmt.Println(".. loading state file")
	appState, err := state.Load()
	if err != nil {
		return err
	}
	defer state.Write(appState)

	client := rostiapi.Client{
		Token:     config.Token,
		CompanyID: appState.CompanyID,
	}

	fmt.Println(".. loading Rostifile")
	rostifile, err := parser.Parse()
	if err != nil {
		return err
	}

	fmt.Printf(".. starting application %s_%d\n", rostifile.Name, appState.ApplicationID)
	err = client.DoApp(appState.ApplicationID, "start")
	if err != nil {
		return err
	}

	fmt.Println(".. all done")

	return nil
}

func commandRemove(c *cli.Context) error {
	config := config.Load()

	fmt.Println(".. loading state file")
	appState, err := state.Load()
	if err != nil {
		return err
	}
	defer state.Write(appState)

	client := rostiapi.Client{
		Token:     config.Token,
		CompanyID: appState.CompanyID,
	}

	fmt.Println(".. loading Rostifile")
	rostifile, err := parser.Parse()
	if err != nil {
		return err
	}

	fmt.Printf(".. removing application %s_%d\n", rostifile.Name, appState.ApplicationID)
	err = client.DeleteApp(appState.ApplicationID)
	if err != nil {
		return err
	}

	fmt.Println(".. removing .rosti.state file")

	err = os.Remove(".rosti.state")
	if err != nil {
		return err
	}

	fmt.Println(".. all done!")

	return nil
}

func commandStatus(c *cli.Context) error {
	config := config.Load()

	fmt.Println(".. loading state file")
	appState, err := state.Load()
	if err != nil {
		return err
	}
	defer state.Write(appState)

	client := rostiapi.Client{
		Token:     config.Token,
		CompanyID: appState.CompanyID,
	}

	fmt.Println(".. loading application status")
	status, err := client.GetAppStatus(appState.ApplicationID)
	if err != nil {
		return err
	}

	app, err := client.GetApp(appState.ApplicationID)
	domains := app.Domains

	fmt.Println()
	printAppStatus(domains, status, app)

	return nil
}

func commandPlans(c *cli.Context) error {
	config := config.Load()

	client := rostiapi.Client{
		Token: config.Token,
	}

	plans, err := client.GetPlans()
	if err != nil {
		return err
	}

	fmt.Printf("  %12s  Plan\n", "Slug")
	fmt.Printf("  %12s  ------------\n", "------------")
	for _, plan := range plans {
		fmt.Printf("  %12s  %s\n", strings.ToLower(plan.Name), plan.Name)
	}
	fmt.Println("")
	fmt.Println("Note: Use slug in your Rostifile.")

	return nil
}

func commandCompanies(c *cli.Context) error {
	config := config.Load()

	client := rostiapi.Client{
		Token: config.Token,
	}

	companies, err := client.GetCompanies()
	if err != nil {
		return err
	}

	fmt.Printf("  %6s  Company name\n", "ID")
	fmt.Printf("  %6s  ------------\n", "------")
	for _, company := range companies {
		fmt.Printf("  %6s  %s\n", strconv.Itoa(int(company.ID)), company.Name)
	}

	return nil
}

func commandRuntimes(c *cli.Context) error {
	config := config.Load()

	client := rostiapi.Client{
		Token: config.Token,
	}

	runtimes, err := client.GetRuntimes()
	if err != nil {
		return err
	}

	fmt.Printf("  Runtime\n")
	fmt.Printf("  ---------------------------\n")
	for _, runtime := range runtimes {
		fmt.Printf("  %s\n", runtime.Image)
	}

	return nil
}
