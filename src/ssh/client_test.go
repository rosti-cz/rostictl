package ssh

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Password: test
var testKeySecured string = `-----BEGIN OPENSSH PRIVATE KEY-----
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
WTdiSeZrE5R9q2ZQC+ZUYY57He0=
-----END OPENSSH PRIVATE KEY-----
`

func TestClient(t *testing.T) {
	err := ioutil.WriteFile("_testkey", []byte(testKeySecured), 0755)
	assert.Nil(t, err)

	client := Client{
		Username:   "root",
		Server:     "localhost",
		Port:       3222,
		SSHKeyPath: "./_testkey",
		Passphrase: []byte("test"),
	}

	_, err = client.client()
	assert.Nil(t, err)
}
