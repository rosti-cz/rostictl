package state

// RostiState holds state of the local project. It keeps info about assigned ID.
type RostiState struct {
	ApplicationID uint `yaml:"app_id"`
	CompanyID     uint `yaml:"company_id"`
}
