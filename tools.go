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

			if info.IsDir() {
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
// The function returns paths to private key, public key and error
func findSSHKey() (string, string, error) {
	keyFileNames := []string{
		// "id_ed25519", // Not supported by current dropbear version
		"id_rsa",
	}

	user, err := user.Current()
	if err != nil {
		return "", "", fmt.Errorf("getting user info error: %w", err)
	}

	for _, keyFilename := range keyFileNames {
		privateKeyPath := path.Join(user.HomeDir, ".ssh", keyFilename)
		publicKeyPath := path.Join(user.HomeDir, ".ssh", keyFilename+".pub")

		_, errPrivate := os.Stat(privateKeyPath)
		_, errPublic := os.Stat(publicKeyPath)

		if !os.IsNotExist(errPrivate) && !os.IsNotExist(errPublic) {
			return privateKeyPath, publicKeyPath, nil
		}
	}

	return "", "", errors.New("no ssh key found")
}

func readLocalSSHPubKey() (string, error) {
	_, publicKeyPath, err := findSSHKey()
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// findCompany returns company ID based the environment
func findCompany(client *rostiapi.Client, appState *state.RostiState, c *cli.Context) (uint, error) {
	companies, err := client.GetCompanies()
	if err != nil {
		return 0, err
	}

	if len(companies) == 0 {
		return 0, errors.New("no company found")
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
			fmt.Println("")
			return companyID, nil
		} else {
			return companyID, errors.New("no company found")
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
		return companyID, errors.New("selected company (" + strconv.Itoa(int(companyIDFromFlag)) + ") not found")
	}

	return companyID, nil
}

// Selects plan based on Rostifile or default settings
func selectPlan(client *rostiapi.Client, rostifile *parser.Rostifile) (uint, error) {
	// TODO: implement something like default plan loaded from the API (needs support in the API)
	if rostifile.Plan == "" {
		rostifile.Plan = "start+"
	}

	fmt.Println(".. loading list of available plans")
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
	fmt.Println(".. loading list of available runtimes")
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

func printAppStatus(domains []string, status rostiapi.AppStatus, app rostiapi.App) {
	fmt.Println("The application is available on these domains:")
	fmt.Println("")
	for _, domain := range domains {
		fmt.Println("   * http://" + domain)
	}
	if len(app.SSHAccess) > 0 {
		fmt.Println("")
		fmt.Println("")
		fmt.Println("SSH access:")
		fmt.Println("")
		fmt.Printf("  SSH command: ssh -p %d %s@%s\n", app.SSHAccess[0].Port, app.SSHAccess[0].Username, app.SSHAccess[0].Hostname)
		fmt.Printf("  SSH URI: ssh://%s@%s:%d\n", app.SSHAccess[0].Username, app.SSHAccess[0].Hostname, app.SSHAccess[0].Port)
	}

	fmt.Println("")
	fmt.Println("")
	fmt.Println("Current status:")
	fmt.Println("")
	if status.Running {
		fmt.Println("  Container: running")
	} else {
		fmt.Println("  Container: NOT running")
	}

	fmt.Printf("    Memory: %.2f / %.2f MB\n", status.Memory.Usage, status.Memory.Limit)
	if status.Storage.Usage >= 0 {
		fmt.Printf("    Storage: %.2f / %.2f GB (over limit: %.2f GB)\n", status.Storage.Usage, status.Storage.Limit, status.Storage.OverLimit)
	} else {
		fmt.Printf("    Storage: - / %.2f GB (over limit: %.2f GB)\n", status.Storage.Limit, status.Storage.OverLimit)
	}

	if status.DNSStatus {
		fmt.Println("  DNS: all good")
	} else {
		fmt.Println("  DNS: records are not set properly or they haven't been propagated to the internet yet")
	}

	if status.HTTPStatus {
		fmt.Println("  HTTP: all good")
	} else {
		fmt.Println("  HTTP: application doesn't return 200-like status code")
	}

	if len(status.Errors) > 0 {
		fmt.Println("")
		fmt.Println("")
		fmt.Println("Error messages:")
		for _, message := range status.Errors {
			fmt.Println("  * " + message)
		}
	}

	if len(status.Info) > 0 {
		fmt.Println("")
		fmt.Println("")
		fmt.Println("Info messages:")
		fmt.Println("")
		for _, message := range status.Info {
			fmt.Println("  * " + message)
		}
	}
	fmt.Println("")
}
