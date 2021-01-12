package parser

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const rostiFilePath = "./Rostifile"

// Parse returns parsed Rostifile
func Parse() (*Rostifile, error) {
	rostifile := Rostifile{}

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
