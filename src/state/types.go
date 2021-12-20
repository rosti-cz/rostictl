package state

// RostiState holds state of the local project. It keeps info about assigned ID.
type RostiState struct {
	ApplicationID uint   `yaml:"app_id"`
	CompanyID     uint   `yaml:"company_id"`
	SSHKeyPath    string `yaml:"ssh_key_path"`
}

func (r *RostiState) SSHPublicKeyPath() string {
	return r.SSHKeyPath + ".pub"
}
