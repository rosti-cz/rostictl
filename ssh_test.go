package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const SSHKeysTestDirectory = "contrib/test_ssh_keys/"

func TestGetLocalSSHKeys(t *testing.T) {
	keys, err := getLocalSSHKeys(SSHKeysTestDirectory)

	assert.Nil(t, err)
	assert.Contains(t, keys, "id_rsa")
	assert.Contains(t, keys, "id_ed25519")
}
