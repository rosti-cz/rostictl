package ssh

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Client encapsulates all SSH methods
type Client struct {
	Username   string
	Server     string
	Port       int
	SSHKeyPath string
	Passphrase []byte
}

func (c *Client) loadSSHKey() []byte {
	content, err := ioutil.ReadFile(c.SSHKeyPath)
	if err != nil {
		panic(err)
	}
	return content
}

// IsKeyPasswordProtected return true if password is needed to use the key
func (c *Client) IsKeyPasswordProtected() bool {
	_, err := ssh.ParsePrivateKey(c.loadSSHKey())
	if err != nil {
		return strings.Contains(err.Error(), "this private key is passphrase protected")
	}
	return false
}

// IsPasswordOk return true if the passphrase match the passphrase for selected SSH key
func (c *Client) IsPasswordOk() (bool, error) {
	_, err := ssh.ParsePrivateKeyWithPassphrase(c.loadSSHKey(), c.Passphrase)
	if err != nil && strings.Contains(err.Error(), "decryption password incorrect") {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *Client) client() (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod
	signer, err := ssh.ParsePrivateKey(c.loadSSHKey())

	// If the key is password protected we ask for the password.
	if err != nil && strings.Contains(err.Error(), "this private key is passphrase protected") {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(c.loadSSHKey(), c.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("loading ssh key error: %v", err)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("loading ssh key error: %v", err)
	}
	authMethods = append(authMethods, ssh.PublicKeys(signer))

	var supportedCiphers = []string{
		"aes128-ctr", "aes192-ctr", "aes256-ctr",
		"aes128-gcm@openssh.com",
		"arcfour256", "arcfour128",
		"twofish256-cbc",
		"twofish-cbc",
		"twofish128-cbc",
		"blowfish-cbc",
		"3des-cbc",
		"arcfour",
		"cast128-cbc",
		"aes256-cbc",
		"aes128-cbc",
	}

	var sshConfig ssh.Config
	sshConfig.SetDefaults()
	sshConfig.Ciphers = supportedCiphers

	sshClientConfig := &ssh.ClientConfig{
		User:            c.Username,
		Auth:            authMethods,
		Config:          sshConfig,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: this could actually open MinM attack vector
	}

	// open SSH connection
	// ssh app@alpha-node-4.rosti.cz -p 12360
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Server, c.Port), sshClientConfig)

	return client, err
}

// SendFile uploads a content into a remote path.
func (c *Client) SendFile(path string, content string) error {
	client, err := c.client()
	if err != nil {
		return err
	}
	defer client.Close()

	// open an SFTP session over an existing ssh connection.
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// Open the file
	f, err := sftpClient.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write([]byte(content))

	return err
}

// Run runs a command on the remote server.
func (c *Client) Run(command string) (*bytes.Buffer, error) {
	client, err := c.client()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// Get session
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var stdouterr bytes.Buffer

	session.Stderr = &stdouterr
	session.Stdout = &stdouterr

	err = session.Run(command)
	return &stdouterr, err
}

// StreamFile streams local file to remote server.
func (c *Client) StreamFile(path string, stream io.Reader) error {
	client, err := c.client()
	if err != nil {
		return fmt.Errorf("loading client error: %s", err)
	}
	defer client.Close()

	// Get session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("creating SSH session error: %s", err)
	}
	defer session.Close()

	writer, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("connecting stdin pipe error: %s", err)
	}
	defer writer.Close()

	err = session.Start("/bin/sh -c 'tee \"" + path + "\" > /dev/null'")
	if err != nil {
		return fmt.Errorf("starting command error: %s", err)
	}

	_, err = io.Copy(writer, stream)
	if err != nil {
		return fmt.Errorf("copying data error: %s", err)
	}
	writer.Close()

	err = session.Wait()
	if err != nil {
		return fmt.Errorf("waiting to finish the command error: %s", err)
	}

	return nil
}
