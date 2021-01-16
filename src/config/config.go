package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"regexp"

	"gopkg.in/yaml.v2"
)

var configDirectory string
var configFile string

// Checks if all required paths and files exist and if not, creates them.
func initialChecks() {
	user, err := user.Current()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	configDirectory = path.Join(user.HomeDir, ".config", "rosti")
	configFile = path.Join(user.HomeDir, ".config", "rosti", "config.yml")

	// If config directory doesn't exist, let's create a new one
	err = os.MkdirAll(configDirectory, 0755)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// If config file doesn't exist, let's create a new one
	_, err = os.Stat(configFile)
	if os.IsNotExist(err) {
		err = writeConfig(&Config{})
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}
}

// Config holds configuration of this tool
type Config struct {
	Token string `yaml:"token"`
}

// Writes config file into its path
func writeConfig(config *Config) error {
	body, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(configFile, body, 0600)
	return err
}

// Reads config from the config file
func readConfig() (Config, error) {
	var config Config

	body, err := ioutil.ReadFile(configFile)
	if err != nil {
		return config, fmt.Errorf("error occurred while reading the config file: %w", err)
	}

	err = yaml.Unmarshal(body, &config)
	if err != nil {
		return config, fmt.Errorf("error occurred while decoding the config file: %w", err)
	}

	return config, err
}

// Load returns configuration taken from environment variables.
func Load() *Config {
	initialChecks()

	config, err := readConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(config.Token) == 0 {
		fmt.Println("API token hasn't been found, please go here: https://admin.rosti.cz/settings/profile/regenerate-token/")
		fmt.Println("Find the API token generated for your account and paste it on the following line.")
		fmt.Print("API token: ")

		fmt.Scanln(&config.Token)

		re := regexp.MustCompile(`^[a-zA-Z0-9]{40}$`)
		if !re.MatchString(config.Token) {
			fmt.Println("Error: given token is not valid.")
			os.Exit(2)
		}

		err = writeConfig(&config)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		fmt.Printf("You can change the token later by removing or editing file: %s\n", configFile)
	}

	return &config
}
