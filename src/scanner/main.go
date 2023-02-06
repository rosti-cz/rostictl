package scanner

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/rosti-cz/cli/src/parser"
)

/*
This package looks into a project directory and tries to figure out how
to deploy it into Rosti. Also it tells the user what to do to make it
compatible.
*/

// Scan checks given directory, asks user a few questions and returns a few bits of configuration for Rostifile.
func Scan(directory string) (RostifileBits, error) {
	var bits RostifileBits
	var err error

	var tech string

	fmt.Println("Rosti supports these technologies. Which one uses this project?")
	fmt.Println("  1 PHP")
	fmt.Println("  2 Python")
	fmt.Println("  3 Node.js")
	fmt.Println("  4 Binary")
	fmt.Println("Last option can be used for languages producing static binaries or")
	fmt.Println("archives that listens on HTTP port 8080. That can be Golang, C but")
	fmt.Println("also Deno and others.")

	fmt.Print("Which technology this project uses? [1-4]: ")
	fmt.Scanln(&tech)

	if tech == "1" {
		bits, err = php(directory)
		if err != nil {
			return bits, err
		}
	} else if tech == "2" {
		bits, err = python(directory)
		if err != nil {
			return bits, err
		}
	} else if tech == "3" {
		bits, err = node(directory)
		if err != nil {
			return bits, err
		}
	} else if tech == "4" {
		bits, err = binary(directory)
		if err != nil {
			return bits, err
		}
	} else {
		return bits, errors.New("invalid choice")
	}

	return bits, nil
}

func python(directory string) (RostifileBits, error) {
	bits := RostifileBits{
		Technology: "python",
		AppPort:    8080,
	}

	var wsgiModule string

	fmt.Println("To run python, you need to set here a python module where WSGI application is located.")
	fmt.Println("For example, if your project is Django and is called myproject, the module will be 'myproject.wsgi'.")
	fmt.Println("Simply look for wsgi.py in your project directory.")
	fmt.Println("You can check Django documentation for more details: https://docs.djangoproject.com/en/3.1/howto/deployment/wsgi/")
	fmt.Println("")
	fmt.Println("In Flask, the documentation is located here: https://flask.palletsprojects.com/en/1.1.x/deploying/wsgi-standalone/")
	fmt.Println("And the module would be \"myproject:create_app()\" for the same project name.")
	fmt.Println("")
	fmt.Println("In Bottle you should point the system into the module where \"bottle.default_app()\" is located.")
	fmt.Println("The documentation can be seen here: https://bottlepy.org/docs/dev/deployment.html")
	fmt.Println("")
	fmt.Print("What is your project's WSGI module: ")
	fmt.Scanln(&wsgiModule)

	if len(wsgiModule) == 0 {
		return bits, errors.New("no WSGI module given")
	}

	// Let the user to choose port
	var appPort string
	var err error
	fmt.Print("The port where your application is listening on [8080]: ")
	fmt.Scanln(&appPort)
	if len(appPort) > 0 {
		bits.AppPort, err = strconv.Atoi(appPort)
		if err != nil {
			return bits, fmt.Errorf("invalid port number: %s", appPort)
		}
	}

	bits.Processes = []parser.Process{
		{
			Name:    "app",
			Command: "/srv/venv/bin/gunicorn -u app -g app -b 0.0.0.0:8080 --access-logfile - --error-logfile - --reload " + wsgiModule,
		},
	}

	_, err = os.Stat(path.Join(directory, "requirements.txt"))
	if !os.IsNotExist(err) {
		fmt.Println(".. file requirements.txt found, dependency installation added to before_commands.")
		bits.AfterCommands = append(
			bits.AfterCommands,
			"cd /srv/app && /srv/venv/bin/pip install -r requirements.txt",
		)
	}

	bits.AfterCommands = append(
		bits.AfterCommands,
		"supervisorctl restart app",
	)

	return bits, nil
}

func php(directory string) (RostifileBits, error) {
	bits := RostifileBits{
		Technology: "php",
	}

	_, err := os.Stat(path.Join(directory, "index.php"))
	if os.IsNotExist(err) {
		fmt.Println("Warning: there is no index.php file in " + directory + ". It's not required but Rosti's HTTP check could be failing.")
	}

	bits.Processes = []parser.Process{
		{
			Name:    "app",
			Command: "/srv/bin/primary_tech/php-fpm -F -O -g /srv/run/php-fpm.pid -y /srv/conf/php-fpm/php-fpm.conf",
		},
	}

	return bits, nil
}

func node(directory string) (RostifileBits, error) {
	bits := RostifileBits{
		Technology: "node",
		AppPort:    3000,
	}

	// Let the user to choose port
	var appPort string
	var err error
	fmt.Print("The port where your application is listening on [3000]: ")
	fmt.Scanln(&appPort)
	if len(appPort) > 0 {
		bits.AppPort, err = strconv.Atoi(appPort)
		if err != nil {
			return bits, fmt.Errorf("invalid port number: %s", appPort)
		}
	}

	// Test existence of package.json
	_, err = os.Stat(path.Join(directory, "package.json"))
	if os.IsNotExist(err) {
		return bits, errors.New("package.json has not been found in your project, please create one and don't forget to add start command")
	}

	packageJSON := PackageJSON{}
	body, err := ioutil.ReadFile(path.Join(directory, "package.json"))
	if err != nil {
		return bits, err
	}
	err = json.Unmarshal(body, &packageJSON)
	if err != nil {
		return bits, err
	}
	if _, ok := packageJSON.Scripts["start"]; !ok {
		return bits, errors.New("start script cannot be found in package.json, make sure that `scripts` field contains a start script that starts HTTP server of your application")
	}

	bits.Processes = []parser.Process{
		{
			Name:    "app",
			Command: "/srv/bin/primary_tech/npm start",
		},
	}
	bits.AfterCommands = []string{
		"cd /srv/app && npm install",
		"supervisorctl restart app",
	}

	return bits, nil
}

// Binary is binary file of a program than listens on port 8000.
func binary(directory string) (RostifileBits, error) {
	bits := RostifileBits{
		Technology: "binary",
	}

	var binaryFile string

	fmt.Println("To setup your application we need to know where is the binary file in directory \"" + directory + "\".")
	if len(directory) == 0 {
		directory = "./"
	} else if directory[len(directory)-1] != '/' {
		directory += "/"
	}
	fmt.Print("Binary file path: " + directory)
	fmt.Scanln(&binaryFile)

	_, err := os.Stat(path.Join(directory, binaryFile))
	if os.IsNotExist(err) {
		return bits, errors.New("given binary filename doesn't exist")
	}

	bits.Processes = []parser.Process{
		{
			Name:    "app",
			Command: path.Join("/srv/app", binaryFile),
		},
	}
	bits.BeforeCommands = []string{
		"supervisorctl stop app",
	}
	bits.AfterCommands = []string{
		"chmod 755 " + path.Join("/srv/app", binaryFile),
		"supervisorctl start app",
	}

	return bits, nil
}
