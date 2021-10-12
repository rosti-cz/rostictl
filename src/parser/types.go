package parser

import (
	"errors"
	"os"
	"regexp"
)

// Process tells the code what to run in background
type Process struct {
	Name            string `yaml:"name"`
	Command         string `yaml:"command"`
	StopKillAsGroup bool   `yaml:"stop_kill_as_group,omitempty"`
}

// Rostifile is structure that keeps info about desired application.
type Rostifile struct {
	Name string `yaml:"name"`
	// Runtime image of the application, default is defined in the backend, usually the latest.
	Runtime string `yaml:"runtime,omitempty"`
	// Primary technology configured in application's container
	Technology string `yaml:"technology,omitempty"`
	// List of domains configured on the load balancer for this application
	Domains []string `yaml:"domains,omitempty"`
	// Enable/Disable HTTPS for all domains
	HTTPS bool `yaml:"https"`
	// Directory with the source code that will be uploaded onto server into /srv/app. Default is .
	SourcePath string `yaml:"source_path,omitempty"`
	// Plan of the service, possible values are: static,start,start+,normal,normal+,pro,pro+,business,business+. Default is defined in the backend.
	Plan string `yaml:"plan,omitempty"`
	// List of background processes running in supervisor
	Processes []Process `yaml:"processes,omitempty"`
	// Crontab jobs
	Crontabs []string `yaml:"crontabs,omitempty"`
	// Commands to run before deploy begins.
	BeforeCommands []string `yaml:"before_commands,omitempty"`
	// Commands to run after deploy ends.
	AfterCommands []string `yaml:"after_commands,omitempty"`
	// Commmands to runs when the application is created
	InitialCommands []string `yaml:"initial_commands,omitempty"`
	// What directories and files to exclude from the deploy
	Exclude []string `yaml:"exclude,omitempty"`
	// Map of files where key is path to the file (including /srv) and value is content of the file
	Files map[string]string `yaml:"files,omitempty"`
}

// Validate runs static validation over the structure and sets defaults values when possible.
func (r *Rostifile) Validate() []error {
	errs := []error{}

	// Set up default source
	if r.SourcePath == "" {
		r.SourcePath = "."
	}

	info, err := os.Stat(r.SourcePath)
	if os.IsNotExist(err) {
		errs = append(errs, errors.New("directory set in source_path doesn't exist"))
	} else if !info.IsDir() {
		errs = append(errs, errors.New("\""+r.SourcePath+"\" in source_path is not a directory"))
	}

	// Name validation, the most important one
	re, err := regexp.Compile("^[a-zA-Z0-9_\\.]*$")
	if err != nil {
		errs = append(errs, err)
	}

	if !re.MatchString(r.Name) {
		errs = append(errs, errors.New("name can contain only these characters: a-zA-Z0-9._"))
	}

	// Processes validation
	re, err = regexp.Compile("^[a-zA-Z0-9_]*$")
	if err != nil {
		errs = append(errs, err)
	}

	for _, process := range r.Processes {
		if !re.MatchString(process.Name) {
			errs = append(errs, errors.New("name can contain only these characters: a-zA-Z0-9_"))
		}
	}

	// Technology validation
	validTechs := []string{
		"python",
		"node",
		"php",
		"",
	}
	var found bool
	for _, validTech := range validTechs {
		if validTech == r.Technology {
			found = true
			break
		}
	}
	if !found {
		errs = append(errs, errors.New("only valid technologies are python, node, php and empty string"))
	}

	return errs
}
