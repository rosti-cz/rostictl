package parser

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const rostiFilePath = "./Rostifile"

// Parse returns parsed Rostifile
func Parse() (*Rostifile, error) {
	rostifile := Rostifile{}
	rostifile.Validate()

	body, err := ioutil.ReadFile(rostiFilePath)
	if err != nil {
		return &rostifile, errors.Wrap(err, "Rostifile reading error")
	}

	err = yaml.Unmarshal(body, &rostifile)
	if err != nil {
		return &rostifile, errors.Wrap(err, "Rostifile parsing error")
	}

	return &rostifile, nil
}

// Init create a new Rostifile in the current working directory
func Init() (Rostifile, error) {
	rostifile := Rostifile{}

	fmt.Print("Name of the project: ")
	fmt.Scanln(&rostifile.Name)
	fmt.Print("In what sub/directory is the project located [.]: ")
	fmt.Scanln(&rostifile.SourcePath)

	validationErrors := rostifile.Validate()
	if len(validationErrors) > 0 {
		fmt.Println("The input is not valid:")
		for _, err := range validationErrors {
			fmt.Println("  " + err.Error())
		}
		os.Exit(2)
	}

	err := Write(rostifile)
	if err != nil {
		return rostifile, err
	}

	fmt.Println(".. a new Rostifile has been created.")

	return rostifile, nil
}

// Write Rostifile
func Write(rostifile Rostifile) error {
	body, err := yaml.Marshal(rostifile)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(rostiFilePath, body, 0644)
	return err
}
