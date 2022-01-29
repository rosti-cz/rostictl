package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rosti-cz/cli/src/config"
	"github.com/rosti-cz/cli/src/parser"
	"github.com/rosti-cz/cli/src/rostiapi"
	"github.com/rosti-cz/cli/src/scanner"
	"github.com/rosti-cz/cli/src/ssh"
	"github.com/rosti-cz/cli/src/state"
	"github.com/urfave/cli/v2"
)

// Import existing application
func commandImport(c *cli.Context) error {
	cYellow.Println(".. loading configuration")
	config := config.Load()

	cYellow.Println(".. setting up API client")
	client := rostiapi.Client{
		Token:      config.Token,
		ExtraError: os.Stderr,
	}

	appState := state.RostiState{}

	// Pick the right company
	cYellow.Println(".. loading list of your companies")
	companyID, err := findCompany(&client, &appState, c)
	if err != nil {
		return err
	}
	appState.CompanyID = companyID
	client.CompanyID = companyID

	appID, err := selectApp(&client)
	if err != nil {
		return err
	}
	appState.ApplicationID = appID
	cGreen.Println(".. done")

	cYellow.Println(".. writing .rosti.state")
	err = state.Write(&appState)
	if err != nil {
		return err
	}

	return nil
}

// Deploys or re-deploys an application
func commandUp(c *cli.Context) error {
	config := config.Load()

	// Rostifile and statefile
	cYellow.Println(".. loading Rostifile")
	rostifile, err := parser.Parse()
	if err != nil {
		return err
	}

	client := rostiapi.Client{
		Token:      config.Token,
		ExtraError: os.Stderr,
	}

	cYellow.Println(".. loading state file")
	appState, err := state.Load()
	if err != nil {
		return err
	}
	defer state.Write(appState)

	// SSH key
	if len(appState.SSHKeyPath) == 0 {
		cYellow.Println(".. SSH key not found in the state file, trying to figure this out")
		privateSSHKeyPath, _, err := findSSHKey()
		if err != nil {
			return fmt.Errorf("SSH key problem: %w", err)
		}
		appState.SSHKeyPath = privateSSHKeyPath
	} else {
		_, err := os.Stat(appState.SSHKeyPath)
		if os.IsNotExist(err) {
			cYellow.Println(".. SSH key configured in state file but the file doesn't not exist, trying to figure this out")
			privateSSHKeyPath, _, err := findSSHKey()
			if err != nil {
				return fmt.Errorf("SSH key problem: %w", err)
			}
			appState.SSHKeyPath = privateSSHKeyPath
		}
	}

	// Pick the right company
	if appState.CompanyID == 0 {
		cYellow.Println(".. loading list of your companies")
		companyID, err := findCompany(&client, appState, c)
		if err != nil {
			return err
		}
		appState.CompanyID = companyID
	}

	client.CompanyID = appState.CompanyID

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
	var appCreated bool // if true the app was just created
	// Update existing app
	if appState.ApplicationID > 0 {
		cYellow.Println(".. loading current state of the application")
		app, err := client.GetApp(appState.ApplicationID)
		if err != nil {
			return err
		}

		// If it's down let's start it
		if !app.Enabled {
			cYellow.Println(".. starting the application because it was off")
			err = client.DoApp(appState.ApplicationID, "start")
			if err != nil {
				return err
			}
		}

		sshPubKey, err := readLocalSSHPubKey(appState.SSHPublicKeyPath())
		if err != nil {
			return err
		}

		// Use update
		cYellow.Printf(".. updating existing application %s_%d \n", rostifile.Name, appState.ApplicationID)

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
		cYellow.Printf(".. creating a new application %s \n", rostifile.Name)

		sshPubKey, err := readLocalSSHPubKey(appState.SSHPublicKeyPath())
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

		appCreated = true
	}

	// SSH client initialization
	if len(newApp.SSHAccess) == 0 {
		return errors.New("no SSH access found")
	}

	sshClient := ssh.Client{
		Server:     newApp.SSHAccess[0].Hostname,
		Port:       int(newApp.SSHAccess[0].Port),
		Username:   newApp.SSHAccess[0].Username,
		SSHKeyPath: appState.SSHKeyPath,
	}

	if sshClient.IsKeyPasswordProtected() {
		passphrase := ""
		fmt.Print("SSH key password: ")
		_, err = fmt.Scanln(&passphrase)
		if err != nil {
			return fmt.Errorf("ssh key password input error: %v", err)
		}

		sshClient.Passphrase = []byte("password")
	}

	// Test SSH connection
	cYellow.Println(".. waiting for SSH daemon to get ready")
	testCounter := 0
	for {
		_, err := sshClient.Run("echo 1")
		if err == nil {
			cGreen.Println("     ready")
			break
		}

		if testCounter > 12 {
			// This prints the last error that occurs. It turned out the problem can actually be
			// something else because this is the first time we try to connect to the SSH server.
			cRed.Println(err.Error())
			return errors.New("SSH daemon has not started in time")
		}

		testCounter++

		time.Sleep(5 * time.Second)
	}

	// Setup technology
	if appCreated { // This is processes only when app is freshly created
		// Call rosti.sh to setup environment for selected technology
		if len(rostifile.Technology) > 0 {
			cRed.Println(".. deleting default code")
			// Clean /srv/app and clean /srv/conf/supervisor.d/app.conf because we don't want the default application
			cmd := "/bin/sh -c 'rm -rf /srv/app/* && rm -rf /srv/conf/supervisor.d/app.conf && supervisorctl reread && supervisorctl update'"
			buf, err := sshClient.Run(cmd)
			if err != nil {
				cYellow.Print("Command '")
				cWhite.Print(cmd)
				cYellow.Println("' error:")
				fmt.Println(buf.String())
				return err
			}
		}
	}

	cYellow.Println(".. loading application status")
	status, err := client.GetAppStatus(appState.ApplicationID)
	if err != nil {
		return fmt.Errorf("GetAppStatus error: %v", err)
	}

	var buf *bytes.Buffer

	if status.PrimaryTech.Name != rostifile.Technology || (status.PrimaryTech.Version != rostifile.TechnologyVersion && rostifile.TechnologyVersion != "") {
		cYellow.Print(".. technology change detected, settings up ")
		cWhite.Print(rostifile.Technology)
		cYellow.Println(" environment")
		cmd := "/usr/local/bin/rosti " + rostifile.Technology

		if rostifile.TechnologyVersion != "" {
			cmd = "/usr/local/bin/rosti " + rostifile.Technology + " " + rostifile.TechnologyVersion
		}

		buf, err = sshClient.Run(cmd)
		if err != nil {
			fmt.Print("Command '")
			cWhite.Print(cmd)
			cYellow.Println("' error:")
			cRed.Println(buf.String())
			return err
		}
	}

	// Initial commands
	if appCreated || c.Bool("force-init") {
		for _, cmd := range rostifile.InitialCommands {
			buf, err := sshClient.Run(cmd)
			if err != nil {
				fmt.Print("Command '")
				cWhite.Print(cmd)
				cYellow.Println("' error:")
				cRed.Println(buf.String())
				return err
			}
		}
	}

	// Deploy files
	cYellow.Println(".. creating an archive")
	err = createArchive(rostifile.SourcePath, "/tmp/_archive.tar", rostifile.Exclude) // TODO: create a proper temporary file here
	if err != nil {
		return err
	}

	cYellow.Println(".. copying archive into the container")
	archive, err := os.Open("/tmp/_archive.tar")
	if err != nil {
		return err
	}
	defer archive.Close()

	err = sshClient.StreamFile("/srv/_archive.tar", archive)
	if err != nil {
		return err
	}

	for _, cmd := range rostifile.BeforeCommands {
		cYellow.Print(".. running command:")
		cWhite.Println(cmd)
		buf, err = sshClient.Run("/bin/sh -c '" + cmd + "'")
		if err != nil {
			fmt.Print("Command '")
			cWhite.Print(cmd)
			cYellow.Println("' error:")
			cRed.Println(buf.String())
			return err
		}
	}

	cYellow.Println(".. unarchiving code in the container")
	cmd := "/bin/sh -c \"mkdir -p /srv/app && mv _archive.tar /srv/app/ && cd /srv/app && tar xf _archive.tar && rm _archive.tar\""
	buf, err = sshClient.Run(cmd)
	if err != nil {
		cRed.Println("Unarchiving error. Command output:")
		cRed.Println(buf.String())
		return err
	}

	for _, cmd := range rostifile.AfterCommands {
		fmt.Printf(".. running command: %s\n", cmd)
		buf, err = sshClient.Run("/bin/sh -c '" + cmd + "'")
		if err != nil {
			fmt.Print("Command '")
			cWhite.Print(cmd)
			cYellow.Println("' error:")
			cRed.Println(buf.String())
			return err
		}
	}

	// Setup crontab
	if len(rostifile.Crontabs) > 0 {
		cYellow.Println(".. setting up crontabs")
		err = sshClient.SendFile("/srv/conf/crontab", strings.Join(rostifile.Crontabs, "\n")+"\n")
		if err != nil {
			return fmt.Errorf("uploading crontabs error: %w", err)
		}
		_, err = sshClient.Run("crontab /srv/conf/crontab")
		if err != nil {
			return fmt.Errorf("refreshing crontab error: %w", err)
		}
	}

	// Setup background processes
	if len(rostifile.Processes) > 0 {
		cYellow.Println(".. setting up supervisor processes")
		var processes []string
		for _, process := range rostifile.Processes {
			processTemplate := `[program:` + process.Name + `]
command=` + process.Command + `
environment=PATH="/srv/bin/primary_tech:/usr/local/bin:/usr/bin:/bin:/srv/.npm-packages/bin"
autostart=true
autorestart=true
directory=/srv/app
process_name=` + process.Name + `
stdout_logfile=/srv/log/` + process.Name + `.log
stdout_logfile_maxbytes=2MB
stdout_logfile_backups=5
stdout_capture_maxbytes=2MB
stdout_events_enabled=false
redirect_stderr=true
`
			if process.StopKillAsGroup {
				processTemplate += "stopasgroup=true\n"
				processTemplate += "killasgroup=true\n"
			}

			processes = append(processes, processTemplate)
		}

		err = sshClient.SendFile("/srv/conf/supervisor.d/rostictl.conf", "# This file is gonna be rewritten by rostictl\n\n"+strings.Join(processes, "\n")+"\n")
		if err != nil {
			return fmt.Errorf("updating supervisor config error: %w", err)
		}

		_, err = sshClient.Run("supervisorctl reread")
		if err != nil {
			return fmt.Errorf("refreshing supervisor error: %w", err)
		}
		_, err = sshClient.Run("supervisorctl update")
		if err != nil {
			return fmt.Errorf("updating supervisor error: %w", err)
		}
	}

	// Done
	cYellow.Println(".. all done, let's check status of the application")

	// Check app's status
	cYellow.Println(".. loading application status")
	status, err = client.GetAppStatus(appState.ApplicationID)
	if err != nil {
		return err
	}

	cYellow.Println(".. loading application configuration")
	app, err := client.GetApp(appState.ApplicationID)
	if err != nil {
		return err
	}

	fmt.Println("")
	printAppStatus(app.Domains, status, app, false)

	fmt.Println("")
	fmt.Println("Note: This output doesn't have to be precise, because container")
	fmt.Println("hasn't had to boot up fully or DNS hasn't propagated into the world.")
	fmt.Println("Run `rostictl status` to run these checks again later to find out what's")
	fmt.Println("the status of this application.")

	return nil
}

func commandDown(c *cli.Context) error {
	config := config.Load()

	cYellow.Println(".. loading state file")
	appState, err := state.Load()
	if err != nil {
		return err
	}
	defer state.Write(appState)

	client := rostiapi.Client{
		Token:      config.Token,
		CompanyID:  appState.CompanyID,
		ExtraError: os.Stderr,
	}

	cYellow.Println(".. loading Rostifile")
	rostifile, err := parser.Parse()
	if err != nil {
		return err
	}

	cYellow.Printf(".. stopping application %s_%d\n", rostifile.Name, appState.ApplicationID)
	err = client.DoApp(appState.ApplicationID, "stop")
	if err != nil {
		return err
	}

	cGreen.Println(".. all done!")

	return nil
}

func commandStart(c *cli.Context) error {
	config := config.Load()

	cYellow.Println(".. loading state file")
	appState, err := state.Load()
	if err != nil {
		return err
	}
	defer state.Write(appState)

	client := rostiapi.Client{
		Token:      config.Token,
		CompanyID:  appState.CompanyID,
		ExtraError: os.Stderr,
	}

	cYellow.Println(".. loading Rostifile")
	rostifile, err := parser.Parse()
	if err != nil {
		return err
	}

	cYellow.Print(".. starting application ")
	cWhite.Println(fmt.Sprintf("%s_%d", rostifile.Name, appState.ApplicationID))
	err = client.DoApp(appState.ApplicationID, "start")
	if err != nil {
		return err
	}

	cGreen.Println(".. all done")

	return nil
}

func commandRemove(c *cli.Context) error {
	config := config.Load()

	cYellow.Println(".. loading state file")
	appState, err := state.Load()
	if err != nil {
		return err
	}

	client := rostiapi.Client{
		Token:     config.Token,
		CompanyID: appState.CompanyID,
	}

	cYellow.Println(".. loading Rostifile")
	rostifile, err := parser.Parse()
	if err != nil {
		return err
	}

	cRed.Print(".. removing application ")
	cWhite.Println(fmt.Sprintf("%s_%d\n", rostifile.Name, appState.ApplicationID))
	err = client.DeleteApp(appState.ApplicationID)
	if err != nil {
		return err
	}

	cRed.Println(".. removing .rosti.state file")

	err = state.Remove()
	if err != nil {
		return err
	}

	cGreen.Println(".. all done!")

	return nil
}

func commandStatus(c *cli.Context) error {
	config := config.Load()

	cYellow.Println(".. loading state file")
	appState, err := state.Load()
	if err != nil {
		return err
	}

	client := rostiapi.Client{
		Token:      config.Token,
		CompanyID:  appState.CompanyID,
		ExtraError: os.Stderr,
	}

	cYellow.Println(".. loading application status")
	status, err := client.GetAppStatus(appState.ApplicationID)
	if err != nil {
		return fmt.Errorf("GetAppStatus error: %v", err)
	}

	app, err := client.GetApp(appState.ApplicationID)
	if err != nil {
		return fmt.Errorf("GetApp error: %v", err)
	}
	domains := app.Domains

	fmt.Println()
	printAppStatus(domains, status, app, true)

	return nil
}

func commandPlans(c *cli.Context) error {
	config := config.Load()

	client := rostiapi.Client{
		Token:      config.Token,
		ExtraError: os.Stderr,
	}

	plans, err := client.GetPlans()
	if err != nil {
		return err
	}

	cGrey.Printf("  %12s  Plan\n", "Slug")
	cGrey.Printf("  %12s  ------------\n", "------------")
	for _, plan := range plans {
		fmt.Printf("  %12s  %s\n", cYellow.Sprint(strings.ToLower(plan.Name)), cGrey.Sprint(plan.Name))
	}
	fmt.Println("")
	fmt.Println("Note: Use slug in your Rostifile.")

	return nil
}

func commandCompanies(c *cli.Context) error {
	config := config.Load()

	client := rostiapi.Client{
		Token:      config.Token,
		ExtraError: os.Stderr,
	}

	companies, err := client.GetCompanies()
	if err != nil {
		return err
	}

	cGrey.Printf("  %6s  Company name\n", "ID")
	cGrey.Printf("  %6s  ------------\n", "------")
	for _, company := range companies {
		fmt.Printf("  %6s  %s\n", cYellow.Sprint(strconv.Itoa(int(company.ID))), cWhite.Sprint(company.Name))
	}

	return nil
}

func commandRuntimes(c *cli.Context) error {
	config := config.Load()

	client := rostiapi.Client{
		Token:      config.Token,
		ExtraError: os.Stderr,
	}

	runtimes, err := client.GetRuntimes()
	if err != nil {
		return err
	}

	cGrey.Printf("  Runtime\n")
	cGrey.Printf("  ---------------------------\n")
	for _, runtime := range runtimes {
		if runtime.Default {
			cGreen.Printf(" *%s\n", runtime.Image)
		} else {
			cYellow.Printf("  %s\n", runtime.Image)
		}
	}

	return nil
}

func commandInit(c *cli.Context) error {
	_, err := os.Stat("./Rostifile")
	if !os.IsNotExist(err) {
		cRed.Println("Rostifile already exists in this directory")
		os.Exit(1)
	}

	rostifile, err := parser.Init()
	if err != nil {
		return err
	}

	bits, err := scanner.Scan(rostifile.SourcePath)
	if err != nil {
		return err
	}

	rostifile.AfterCommands = bits.AfterCommands
	rostifile.BeforeCommands = bits.BeforeCommands
	rostifile.Processes = bits.Processes
	rostifile.Technology = bits.Technology

	err = parser.Write(rostifile)
	if err != nil {
		return err
	}

	return nil
}

func commandVersion(c *cli.Context) error {
	fmt.Println("Version:", version)
	return nil
}
