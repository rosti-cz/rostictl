package ssh

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Client encapsulates all SSH methods
type Client struct {
	Username   string
	Server     string
	Port       int
	SSHKeyPath string
}

func (c *Client) loadSSHKey() []byte {
	content, err := ioutil.ReadFile(c.SSHKeyPath)
	if err != nil {
		panic(err)
	}
	return content
}

func (c *Client) client() (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod
	signer, err := ssh.ParsePrivateKey(c.loadSSHKey())
	if err != nil {
		return nil, err
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
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// open SSH connection
	// ssh app@alpha-node-4.rosti.cz -p 12360
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Server, c.Port), sshClientConfig)

	return client, err
}

// sendFile uploads a content into a remote path.
func (c *Client) sendFile(server string, path string, content string) error {
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
