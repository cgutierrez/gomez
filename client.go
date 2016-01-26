package gomez

import (
  "io/ioutil"
  "log"
  "fmt"
  "os/user"

  "golang.org/x/crypto/ssh"
  "github.com/gcmurphy/getpass"
)

func LoadKeyFile(path string) (ssh.Signer, error) {
  var signer ssh.Signer

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
    log.Fatalln(err)
  }

  files := []ssh.Signer {}

  var idRsa ssh.Signer
  if idRsa, err = LoadKeyFile(usr.HomeDir + "/.ssh/id_rsa"); err != nil {
    return nil, err
  }

  files = append(files, idRsa)

  return files, nil
}

func CreateSession(host *Host) (*ssh.Client, *ssh.Session, error) {

  sshConfig := &ssh.ClientConfig {
    User: host.User,
    Auth: []ssh.AuthMethod {},
  }

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
      password, err :=  getpass.GetPassWithOptions(fmt.Sprintf("enter password for %s:", host.Host), 0, 100)

      if err != nil {
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