package parser

// Rostifile is structure that keeps info about desired application.
type Rostifile struct {
	Name string `yaml:"name"`
	// Runtime image of the application, default is the latest.
	Runtime string `yaml:"runtime"`
	// List of domains configured on the load balancer for this application
	Domains []string `yaml:"domains"`
	// Enable/Disable HTTPS for all domains
	HTTPS bool `yaml:"https"`
	// Type of the application, it can be python,php,node.js,ruby,golang,deno. This is required.
	Type string `yaml:"type"`
	// Directory with the source code that will be uploaded onto server into /srv/app. Default is .
	Source string `yaml:"source"`
	// Plan of the service, possible values are: static,start,start+,normal,normal+,pro,pro+,business,business+. Default is start.
	Plan string `yaml:"plan"`
	// List of background processes running in supervisor
	Processes []string `yaml:"processes"`
	// Crontab jobs
	Crontabs []string `yaml:"crontabs"`
	// Commands to run before deploy begins. Default: supervisorctl stop app
	BeforeCommands []string `yaml:"before_commands"`
	// Commands to run after deploy ends. Default: supervisorctl start app
	AfterCommands []string `yaml:"after_commands"`
}
