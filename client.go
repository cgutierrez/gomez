package gomez

import (
	"fmt"
	"io/ioutil"
	_"log"
	"os/user"
  "strings"

	"github.com/gcmurphy/getpass"
	"golang.org/x/crypto/ssh"
  "github.com/cgutierrez/goshen"
)

var (
  userSshConfig *goshen.SshConfig
)

func init() {
  userSshConfig = goshen.NewSshConfig("~/.ssh/config")
}

func LoadKeyFile(path string) (ssh.Signer, error) {
	var signer ssh.Signer

  // normalize the path when using the users home directory shortcut
  if strings.HasPrefix(path, "~") {
    usr, err := user.Current()

    if err != nil {
      return nil, err
    }

    path = strings.Replace(path, "~", usr.HomeDir, 1)
  }

	privateBytes, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	signer, err = ssh.ParsePrivateKey(privateBytes)

	if err != nil {
		return nil, err
	}

	return signer, err
}

func LoadDefaultKeyFiles() ([]ssh.Signer, error) {

	usr, err := user.Current()

	if err != nil {
		return nil, err
	}

	files := []ssh.Signer{}

	var idRsa ssh.Signer
	if idRsa, err = LoadKeyFile(usr.HomeDir + "/.ssh/id_rsa"); err != nil {
		return nil, err
	}

	files = append(files, idRsa)

	return files, nil
}

func CreateSession(host *Host) (*ssh.Client, *ssh.Session, error) {

  // if the host configuration doesn't contain a key file, check the users .ssh/config file
  // appends any keys found for the matching host to be used for authentication
  foundHostConfig := userSshConfig.MatchHost(host.Host)
  if host.KeyFile == "" && foundHostConfig != nil {

    if foundHostConfig != nil {

      // set the host to the hostname found in the configuration
      // allows for using partial host names in the host argument
      if foundHostConfig.HostName != "" {
        host.Host = foundHostConfig.HostName
      }

      if host.KeyFile == "" && foundHostConfig.IdentityFile != "" {
        host.KeyFile = foundHostConfig.IdentityFile
      }

      // use the port form the ssh config if it's supplied
      if host.Port == "" && foundHostConfig.Port != "" {
        host.Port = foundHostConfig.Port
      }

      // use the user found in the foundHostConfig if one isn't provided
      if host.User == "" && foundHostConfig.User != "" {
        host.User = foundHostConfig.User
      }
    }
  }

	sshConfig := &ssh.ClientConfig { User: host.User, Auth: []ssh.AuthMethod{}, }

	if host.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(host.Password))
	}

	if host.KeyFile != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeysCallback(func() (signers []ssh.Signer, err error) {

			keyFiles, err := LoadDefaultKeyFiles()

			if host.KeyFile != "" {
				hostKeyFile, err := LoadKeyFile(host.KeyFile)
				if err != nil {
					return nil, err
				}

				keyFiles = append(keyFiles, hostKeyFile)
			}

			return keyFiles, err
		}))
	}

	if host.Password == "" && host.KeyFile == "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.PasswordCallback(func() (string, error) {
			password, err := getpass.GetPassWithOptions(fmt.Sprintf("enter password for %s@%s: ", host.User, host.Host), 0, 100)

			if err != nil {
        fmt.Println(err.Error())
				return password, err
			}

			host.Password = password
			return password, err
		}))
	}

	client, err := ssh.Dial("tcp", host.Host + ":" + host.Port, sshConfig)

	if err != nil {
		fmt.Println(err.Error())
		return nil, nil, err
	}

	session, err := client.NewSession()

	if err != nil {
		return nil, nil, err
	}

	return client, session, err
}
