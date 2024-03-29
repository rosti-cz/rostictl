package main

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/rosti-cz/cli/src/parser"
	"github.com/rosti-cz/cli/src/rostiapi"
	"github.com/rosti-cz/cli/src/state"
	"github.com/urfave/cli/v2"
)

func createArchive(source, target string, exclude []string) error {
	tarfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarfile.Close()

	tarball := tar.NewWriter(tarfile)
	defer tarball.Close()

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	return filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			for _, excludedItem := range exclude {
				if info.IsDir() && info.Name() == excludedItem {
					return filepath.SkipDir
				} else if info.Name() == excludedItem {
					return nil
				}
			}

			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
			}

			if err := tarball.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() || !info.Mode().IsRegular() { // IsRegular is added because without it it fails on Mac
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarball, file)
			return err
		})
}

// Returns list of SSH key found in the current system.
// It looks for the keys in ~/.ssh which should be valid for Linux, Mac and possibly Windows.
// The function returns paths to the private key, public key and error if there is any
func findSSHKey() (string, string, error) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("getting user info error: %w", err)
	}

	sshKeysDirectory := path.Join(dirname, ".ssh")

	keysFilenames, err := getLocalSSHKeys(sshKeysDirectory)
	if err != nil {
		return "", "", fmt.Errorf("loading local SSH keys error: %w", err)
	}

	user, err := user.Current()
	if err != nil {
		return "", "", fmt.Errorf("getting user info error: %w", err)
	}

	// Manually entered key
	if len(keysFilenames) == 0 {
		cRed.Println("No local SSH key found. Please enter path to your private key manually: ")
		cYellow.Print("> ")
		var keyPath string
		_, err := fmt.Scanln(&keyPath)
		if err != nil {
			return "", "", fmt.Errorf("reading user input error: %w", err)
		}
		keyPath = strings.Replace(keyPath, "~", user.HomeDir, 1)

		_, err = os.Stat(keyPath)
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("file %s does not exist", keyPath)
		}

		_, err = os.Stat(keyPath + ".pub")
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("file %s.pub does not exist", keyPath)
		}

		return keyPath, keyPath + ".pub", nil
	}

	// If there is only one discovered key
	if len(keysFilenames) == 1 {

	}

	// Select one of the discovered keys
	cWhite.Printf("Following keys were discovered in %s.\n", sshKeysDirectory)
	cWhite.Println("Please select one: ")
	fmt.Println("")

	cGrey.Printf("  %6s  Key name\n", "ID")
	cGrey.Printf("  %6s  ------------\n", "------")
	var index int = 1
	for _, keyFilename := range keysFilenames {
		fmt.Printf("  %6s  %s\n", cYellow.Sprint(strconv.Itoa(index)), cWhite.Sprint(keyFilename))
		index += 1
	}
	fmt.Println("")
	cYellow.Print("> ")

	var selectionRaw string
	_, err = fmt.Scanln(&selectionRaw)
	if err != nil {
		return "", "", fmt.Errorf("reading user input error: %w", err)
	}

	selection, err := strconv.Atoi(selectionRaw)
	if err != nil {
		return "", "", fmt.Errorf("user input error: %w", err)
	}

	if selection < 1 || selection > len(keysFilenames) {
		cRed.Println("ERROR: Invalid key index entered")
		os.Exit(1)
	}

	privateKeyPath := path.Join(user.HomeDir, ".ssh", keysFilenames[selection-1])
	publicKeyPath := path.Join(user.HomeDir, ".ssh", keysFilenames[selection-1]+".pub")

	_, err = os.Stat(privateKeyPath)
	if os.IsNotExist(err) {
		return "", "", fmt.Errorf("file %s does not exist", privateKeyPath)
	}

	_, err = os.Stat(publicKeyPath)
	if os.IsNotExist(err) {
		return "", "", fmt.Errorf("file %s does not exist", publicKeyPath)
	}

	return privateKeyPath, publicKeyPath, nil
}

func readLocalSSHPubKey(publicKeyPath string) (string, error) {
	body, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// findCompany returns company ID based the environment
func findCompany(client *rostiapi.Client, appState *state.RostiState, c *cli.Context) (uint, error) {
	// When company is forced by a parameter
	if c.Int("company") != 0 {
		return uint(c.Int("company")), nil
	}

	// When company is set already
	companyID := appState.CompanyID
	if companyID != 0 {
		return companyID, nil
	}

	// When company is not set yet
	companies, err := client.GetCompanies()
	if err != nil {
		return 0, err
	}

	if len(companies) == 0 {
		return 0, errors.New("no company found")
	}

	if len(companies) == 1 {
		companyID = companies[0].ID
	} else if len(companies) > 1 {
		fmt.Println("You have access to multiple companies, pick one of the companies below:")
		fmt.Println("")
		fmt.Printf("  %6s  Company name\n", "ID")
		fmt.Printf("  %6s  ------------\n", "------")
		for _, company := range companies {
			fmt.Printf("  %6s  %s\n", strconv.Itoa(int(company.ID)), company.Name)
		}
		fmt.Println("")

		fmt.Print("Pick one of the IDs: ")

		var companyIDRaw string
		_, err := fmt.Scanln(&companyIDRaw)
		if err != nil {
			return companyID, fmt.Errorf("scanning user's input error: %v", err)
		}

		companyID_, err := strconv.Atoi(companyIDRaw)
		companyID = uint(companyID_)
		if err != nil {
			return companyID, fmt.Errorf("input error: %v", err)
		}
	} else {
		return companyID, errors.New("no company found")
	}

	var found bool
	for _, company := range companies {
		if company.ID == companyID {
			found = true
			break
		}
	}
	if !found {
		return companyID, errors.New("selected company (" + strconv.Itoa(int(companyID)) + ") not found")
	}

	return companyID, nil
}

// Returns ID of selected application
func selectApp(client *rostiapi.Client) (uint, error) {
	apps, err := client.GetApps()
	if err != nil {
		return 0, fmt.Errorf("listing app error: %v", err)
	}

	var appID uint

	if len(apps) == 0 {
		return appID, errors.New("no app found")
	} else if len(apps) == 1 {
		appID = apps[0].ID
		cYellow.Println("WARNING: Only one application found, selecting that one.")
	} else {
		fmt.Println("Select application to import. There won't be any change to the application\n" +
			"but rosti.state file will be generated for the current working directory and\n" +
			"you will be able to deploy it as selected app.")
		fmt.Println("")
		cGrey.Printf("  %6s  App name\n", "ID")
		cGrey.Printf("  %6s  --------\n", "------")

		for _, app := range apps {
			fmt.Printf("  %6s  %s\n", cYellow.Sprint(strconv.Itoa(int(app.ID))), cWhite.Sprint(app.Name))
		}
		fmt.Println("")

		cWhite.Println("Pick one of the IDs")
		cGreen.Print("> ")

		var appIDRaw string
		_, err := fmt.Scanln(&appIDRaw)
		if err != nil {
			return appID, fmt.Errorf("scanning user's input error: %v", err)
		}

		appID_, err := strconv.Atoi(appIDRaw)
		appID = uint(appID_)
		if err != nil {
			return appID, fmt.Errorf("input error: %v", err)
		}
	}

	var found bool
	for _, app := range apps {
		if app.ID == appID {
			found = true
			break
		}
	}
	if !found {
		return appID, fmt.Errorf("selected app (%d) not found", appID)
	}

	return appID, nil
}

// Selects plan based on Rostifile or default settings
func selectPlan(client *rostiapi.Client, rostifile *parser.Rostifile) (uint, error) {
	// TODO: implement something like default plan loaded from the API (needs support in the API)
	if rostifile.Plan == "" {
		rostifile.Plan = "start+"
	}

	cYellow.Println(".. loading list of available plans")
	plans, err := client.GetPlans()
	if err != nil {
		return 0, err
	}

	var planID uint
	for _, plan := range plans {
		if strings.ToLower(plan.Name) == strings.ToLower(rostifile.Plan) {
			planID = plan.ID
		}
	}

	return planID, nil
}

// Selects runtime image based on rostifile
func selectRuntime(client *rostiapi.Client, rostifile *parser.Rostifile) (string, error) {
	cYellow.Println(".. loading list of available runtimes")
	runtimes, err := client.GetRuntimes()
	if err != nil {
		return "", err
	}

	var selectedRuntime string

	if len(runtimes) == 0 {
		return selectedRuntime, errors.New("no runtime available")
	}

	// Find default runtime if there is non available
	if len(rostifile.Runtime) == 0 {
		for _, runtime := range runtimes {
			if runtime.Default {
				selectedRuntime = runtime.Image
				break
			}
		}
	} else {
		// Check if selected runtime exists
		for _, runtime := range runtimes {
			if runtime.Image == rostifile.Runtime {
				selectedRuntime = rostifile.Runtime
				break
			}
		}
	}

	if selectedRuntime == "" {
		return selectedRuntime, errors.New("no suitable runtime found, check runtimes command to pick one")
	}

	return selectedRuntime, nil
}

func printAppStatus(domains []string, status rostiapi.AppStatus, app rostiapi.App, showTechs bool) {
	cYellow.Println("The application is available on these domains:")
	fmt.Println("")
	for _, domain := range domains {
		fmt.Println("   * http://" + domain)
	}
	if len(app.SSHAccess) > 0 {
		fmt.Println("")
		fmt.Println("")
		cYellow.Println("SSH access:")
		fmt.Println("")
		fmt.Printf("  SSH command: ssh -p %d %s@%s\n", app.SSHAccess[0].Port, app.SSHAccess[0].Username, app.SSHAccess[0].Hostname)
		fmt.Printf("  SSH URI: ssh://%s@%s:%d\n", app.SSHAccess[0].Username, app.SSHAccess[0].Hostname, app.SSHAccess[0].Port)
	}

	fmt.Println("")
	fmt.Println("")
	cYellow.Println("Current status:")
	fmt.Println("")
	if status.Running {
		cGreen.Println("  Container: running")
	} else {
		cRed.Println("  Container: NOT running")
	}

	fmt.Printf("    Memory: %.2f / %.2f MB\n", status.Memory.Usage, status.Memory.Limit)
	if status.Storage.Usage >= 0 {
		fmt.Printf("    Storage: %.2f / %.2f GB (over limit: %.2f GB)\n", status.Storage.Usage, status.Storage.Limit, status.Storage.OverLimit)
	} else {
		fmt.Printf("    Storage: - / %.2f GB (over limit: %.2f GB)\n", status.Storage.Limit, status.Storage.OverLimit)
	}

	if status.DNSStatus {
		cGreen.Println("  DNS: all good")
	} else {
		cYellow.Println("  DNS: records are not set properly or they haven't been propagated to the internet yet")
	}

	if status.HTTPStatus {
		cGreen.Println("  HTTP: all good")
	} else {
		cRed.Println("  HTTP: application doesn't return 200-like status code")
	}

	if len(status.Errors) > 0 {
		fmt.Println("")
		fmt.Println("")
		cYellow.Println("Error messages:")
		for _, message := range status.Errors {
			cRed.Println("  * " + message)
		}
	}

	if len(status.Info) > 0 {
		fmt.Println("")
		fmt.Println("")
		cYellow.Println("Info messages:")
		fmt.Println("")
		for _, message := range status.Info {
			fmt.Println("  * " + message)
		}
	}

	if showTechs {
		fmt.Println("")
		fmt.Println("")
		cYellow.Println("Available technologies:")
		fmt.Println("")

		for _, tech := range status.Techs {
			if tech.Name == status.PrimaryTech.Name && tech.Version == status.PrimaryTech.Version {
				fmt.Printf(cGreen.Sprint("  %-10s %s <--\n"), tech.Name, tech.Version)
			} else {
				fmt.Printf("  %-10s %s\n", tech.Name, tech.Version)
			}
		}
	}

	fmt.Println("")
}

// noColor disables color output
func noColor(c *cli.Context) error {
	if c.Bool("no-color") {
		color.NoColor = true
	}

	return nil
}
