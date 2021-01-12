package state

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

const rostiStateFilePath = "./.rosti.state"

// Load returns parsed RostiState
func Load() (*RostiState, error) {
	rostiStateFile := RostiState{}

	// Create a file if it doesn't exist
	if _, err := os.Stat(rostiStateFilePath); os.IsNotExist(err) {
		err = ioutil.WriteFile(rostiStateFilePath, []byte(""), 0644)
		if err != nil {
			return &rostiStateFile, fmt.Errorf("rosti state file writing error: %w", err)
		}
	}

	// And then read its content and parse it
	body, err := ioutil.ReadFile(rostiStateFilePath)
	if err != nil {
		return &rostiStateFile, fmt.Errorf("rosti state file reading error: %w", err)
	}

	err = yaml.Unmarshal(body, &rostiStateFile)
	if err != nil {
		return &rostiStateFile, fmt.Errorf("rosti state file parsing error: %w", err)
	}

	return &rostiStateFile, nil
}

// Write writes state structure content into a designated path
func Write(state *RostiState) error {
	body, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("rosti state file yaml encoding error: %w", err)
	}

	err = ioutil.WriteFile(rostiStateFilePath, body, 0644)
	if err != nil {
		return fmt.Errorf("rosti state file writing error: %w", err)
	}

	return nil
}
