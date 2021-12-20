package main

import (
	"io/ioutil"
	"os"
	"regexp"
)

// getLocalSSHKeys returns list of keys located in ~/.ssh, this is probably
// Unix only way so we return empty list when on Windows (~/.ssh doesn't exist)
func getLocalSSHKeys(sshKeysDirectory string) ([]string, error) {
	var sshKeys []string

	// Check where the SSH keys directory exists
	_, err := os.Stat(sshKeysDirectory)
	if os.IsNotExist(err) {
		return sshKeys, nil
	}

	files, err := ioutil.ReadDir(sshKeysDirectory)
	if err != nil {
		return sshKeys, err
	}

	for _, file := range files {
		matched, err := regexp.Match("^id_", []byte(file.Name()))
		if err != nil {
			return sshKeys, err
		}

		matchedTail, err := regexp.Match(".pub$", []byte(file.Name()))
		if err != nil {
			return sshKeys, err
		}

		if matched && !matchedTail {
			sshKeys = append(sshKeys, file.Name())
		}
	}

	return sshKeys, err
}
