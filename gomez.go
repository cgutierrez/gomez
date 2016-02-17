package gomez

import (
	_"bufio"
	"fmt"
  _"io"
	_"io/ioutil"
	_"log"
  _"bufio"
	"os/user"
	_"os/exec"
	_"path/filepath"
	_"sync"

	_"golang.org/x/crypto/ssh"
)

var (
  HideCommandOutput bool = false
  localhost *Host = &Host {}
)

type CmdResult struct {
  Result string
  Error error
}

type CmdOptions struct {
  UseSudo bool
  CaptureOutput bool
}

type Host struct {
	User     string
	Password string
	Host     string
	Port     string
	KeyFile  string
}

type ClientConfig struct {
	Hosts            []*Host
	CurrentDirectory string
  localhost *Host
}

func SilenceOutput(beQuiet bool) {
	HideCommandOutput = beQuiet
}

func OutputLocal(message string) {

	if HideCommandOutput || len(message) == 0 {
		return
	}

	fmt.Printf("local: %s\n", message)
}

func OutputRemote(host *Host, message string) {
	if HideCommandOutput || len(message) == 0 {
		return
	}

	fmt.Printf("%s: %s\n", host.Host, message)
}

func CreateHosts(hosts []map[string]string) []*Host {

	mappedHosts := make([]*Host, len(hosts))

	for i, host := range hosts {

		hostConfig := Host{}

		if user, ok := host["user"]; ok && user != "" {
			hostConfig.User = user
		}

		if password, ok := host["password"]; ok && password != "" {
			hostConfig.Password = password
		}

		if keyFile, ok := host["keyFile"]; ok && keyFile != "" {
			hostConfig.KeyFile = keyFile
		}

		hostConfig.Port = "22"
		if port, ok := host["port"]; ok && port != "" {
			hostConfig.Port = port
		}

		if hostName, ok := host["host"]; ok && hostName != "" {
			hostConfig.Host = hostName
		}

		mappedHosts[i] = &hostConfig
	}

	return mappedHosts
}

func NewClient(hosts []*Host) *ClientConfig {

	config := &ClientConfig {}
	config.Hosts = hosts
  config.localhost = localhost

  currentUser, err := user.Current()
  if err != nil {
    panic("could not determine current user")
  }

  localhost.User = currentUser.Username

	return config
}

func NewLocalClient() *ClientConfig {

  currentUser, err := user.Current()
  if err != nil {
    panic("could not determine current user")
  }

  localhost.User = currentUser.Username

	return &ClientConfig { localhost: localhost }
}

func (config *ClientConfig) NewClientInWorkingDirectory(workingDirectory string) *ClientConfig {
	return &ClientConfig { Hosts: config.Hosts, CurrentDirectory: workingDirectory, localhost: localhost }
}