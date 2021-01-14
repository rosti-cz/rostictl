package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"

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
		fmt.Println("Your SSH keys cannot be located.")
		fmt.Println("Please put your public and private RSA or ED25519 keys into:")
		fmt.Println(" ", path.Join(user.HomeDir, ".ssh", "id_X"))
		fmt.Println(" ", path.Join(user.HomeDir, ".ssh", "id_X.pub"))
		fmt.Println("and try again.")
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
			Domain:  rostifile.Domains,
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
			Domain:  rostifile.Domains,
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

	fmt.Println(".. unarchiving code in the container")

	var buf *bytes.Buffer

	for _, cmd := range rostifile.BeforeCommands {
		buf, err = sshClient.Run(cmd)
		if err != nil {
			fmt.Println("Command '" + cmd + "' error:")
			fmt.Println(buf.String())
			return err
		}
	}

	cmd := "/bin/sh -c \"mkdir -p /srv/app && mv _archive.tar /srv/app/ && cd /srv/app && tar xf _archive.tar\""
	buf, err = sshClient.Run(cmd)
	if err != nil {
		fmt.Println("Unarchiving error. Command output:")
		fmt.Println(buf.String())
		return err
	}

	for _, cmd := range rostifile.AfterCommands {
		buf, err = sshClient.Run(cmd)
		if err != nil {
			fmt.Println("Command '" + cmd + "' error:")
			fmt.Println(buf.String())
			return err
		}
	}

	fmt.Println("All done!")
	// TODO: print status of the application that tells user details about the app

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

	fmt.Println("All done!")

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

	fmt.Println("All done!")

	return nil
}
