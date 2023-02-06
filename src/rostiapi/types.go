package rostiapi

import "strings"

// SSHAccess holds access via SSH protocol
type SSHAccess struct {
	Hostname string `json:"hostname"`
	Port     uint   `json:"port"`
	Username string `json:"username"`
}

// App is structure keeping backend data about an application
type App struct {
	ID           uint        `json:"id,omitempty"`
	Date         string      `json:"date,omitempty"`
	Name         string      `json:"name"`
	Enabled      bool        `json:"enabled,omitempty"`
	Image        string      `json:"image,omitempty"`
	Domains      []string    `json:"domains,omitempty"`
	Mode         string      `json:"mode,omitempty"`
	Plan         uint        `json:"plan,omitempty"`
	SSHAccess    []SSHAccess `json:"ssh_access,omitempty"`
	SSHKeys      []string    `json:"ssh_keys,omitempty"`
	SMTPUsername string      `json:"smtp_username,omitempty"`
	SMTPToken    string      `json:"smtp_token,omitempty"`
	AppPort      int         `json:"app_port,omitempty"`
}

// ErrorResponse represents message returned by the API in case of non-200 response
type ErrorResponse struct {
	Message string                 `json:"message"`
	Errors  map[string]interface{} `json:"errors"`
	// TODO: add errors: {"message":"validation error","errors":{"domains":["Toto pole nesmí být prázdné (null)."]}}
}

// Action tells API what to do with the application
type Action struct {
	Action string `json:"action"` // Can be start, stop, restart, rebuild
}

// Plan defines RAW parameters of the service
type Plan struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	RAM      uint   `json:"ram"`
	Disk     uint   `json:"disk"`
	Price    uint   `json:"price"`
	CPUQuote uint   `json:"cpu_quota"`
}

// Company groups people around one project
type Company struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// Runtime is environment where the application is running
type Runtime struct {
	ID      uint   `json:"id"`
	Image   string `json:"image"`
	Default bool   `json:"default"` // default runtime
	Show    bool   `json:"show"`    // shown in admin
}

// AppTech keeps into about a single Runtime's technology and its version
type AppTech struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (a *AppTech) PrintableName() string {
	if a.Name == "node" {
		return "Node.js"
	} else if a.Name == "php" {
		return "PHP"
	}

	return strings.Title(a.Name)
}

// AppStatus contains status information about one application
type AppStatus struct {
	Errors     []string `json:"errors"`
	Info       []string `json:"info"`
	DNSStatus  bool     `json:"dns_status"`
	HTTPStatus bool     `json:"http_status"`
	Running    bool     `json:"running"`
	Storage    struct {
		Usage     float64 `json:"usage"`
		Limit     float64 `json:"limit"`
		OverLimit float64 `json:"over_limit"`
	} `json:"storage"`
	Memory struct {
		Usage float64 `json:"usage"`
		Limit float64 `json:"limit"`
	} `json:"memory"`
	PrimaryTech AppTech   `json:"primary_tech"`
	Techs       []AppTech `json:"techs"`
}
