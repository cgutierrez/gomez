package gomez

import (
  "fmt"
  "os"
  "os/exec"
  "bufio"
  "log"
  "sync"
  "path/filepath"
  "io/ioutil"

  "golang.org/x/crypto/ssh"
)

var HideCommandOutput bool = false

type Host struct {
  User string
  Password string
  Host string
  Port string
  KeyFile string
}

type ClientConfig struct {
  Hosts []Host
  CurrentDirectory string
}

type runner struct {
  Host Host
  ClientConfig *ClientConfig
  Client *ssh.Client
  Session *ssh.Session
  WaitGroup *sync.WaitGroup
  UseSudo bool
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

func OutputRemote(host Host, message string) {
  if HideCommandOutput || len(message) == 0 {
    return
  }

  fmt.Printf("%s: %s\n", host.Host, message)
}


func NewClient(hosts []map[string]string) *ClientConfig {

  config := &ClientConfig {}

  for i := 0; i < len(hosts); i++ {

    hostConfig := Host {}

    host := hosts[i]

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

    config.Hosts = append(config.Hosts, hostConfig)
  }

  return config
}

func (config *ClientConfig) NewClientInWorkingDirectory(workingDirectory string) *ClientConfig {
  return &ClientConfig { Hosts: config.Hosts, CurrentDirectory: workingDirectory }
}

func (config *ClientConfig) Put(source string, destination string) {

  files, err := filepath.Glob(source)
  if err != nil {
    log.Fatalln(err)
  }

  for _, file := range files {

    contents, err := ioutil.ReadFile(file)
    if err != nil {
      log.Fatalln(err)
    }

    fileStat, err := os.Stat(file)
    if err != nil {
      log.Fatalln(err)
    }

    fileMode := fileStat.Mode() & 1023

    var sessionWaitGroup sync.WaitGroup
    sessionWaitGroup.Add(len(config.Hosts) * 3)

    // iterate over the hosts and copy the contents of the file to the remote host
    for _, host := range config.Hosts {
      client, session, err := CreateSession(&host)

      if err != nil {
        log.Fatalln(err)
      }

      OutputLocal(fmt.Sprintf("copy %s to %s:%s", file, host.Host, destination))

      go func(c *ssh.Client, s *ssh.Session, swg *sync.WaitGroup, fileName string, mode os.FileMode) {

        defer swg.Done()
        defer c.Close()
        defer s.Close()

        stdOutReader, err := session.StdoutPipe()

        if err != nil {
          log.Fatalln(err)
        }

        stdErrReader, err := session.StderrPipe()

        if err != nil {
          log.Fatalln(err)
        }

        stdOutScanner := bufio.NewScanner(stdOutReader)
        go func() {
          for stdOutScanner.Scan() {
            OutputRemote(host, stdOutScanner.Text())
          }

          defer sessionWaitGroup.Done()
        }()

        stdErrScanner := bufio.NewScanner(stdErrReader)
        go func() {
          for stdErrScanner.Scan() {
            OutputRemote(host, stdErrScanner.Text())
          }

          defer sessionWaitGroup.Done()
        }()

        w, _ := session.StdinPipe()

        _, lfile := filepath.Split(fileName)
        err = session.Start("/usr/bin/scp -qrt " + destination)
        if err != nil {
          log.Fatalln(err)
        }

        fmt.Fprintf(w, "C%04o %d %s\n", mode, len(contents), lfile)
        w.Write(contents)
        fmt.Fprintf(w, "\x00")
        w.Close()

      }(client, session, &sessionWaitGroup, file, fileMode)

      sessionWaitGroup.Wait()
    }
  }
}

// run a command on the remote hosts
func (config *ClientConfig) Run(cmd string) (string) {
  return config.run(cmd, false)
}

// run a command using sudo on the remote hosts
func (config *ClientConfig) Sudo(cmd string) (string) {
  return config.run(cmd, true)
}

// run a local command
func (config *ClientConfig) Local(cmd string) {
  config.local(cmd, false)
}

// run a local command and return the result as a string
func (config *ClientConfig) LocalWithReturn(cmd string) (string, error) {
  return config.local(cmd, true)
}

func (config *ClientConfig) local(cmd string, captureOutput bool) (string, error) {

  cdCmd := ""
  if config.CurrentDirectory != "" {
    cdCmd = "cd " + config.CurrentDirectory + " && "
  }

  cmdStr := cdCmd + cmd

  execCmd := exec.Command("/bin/sh", "-c", cmdStr)

  // return the output of the command if set to captureOutput
  if captureOutput {
    cmdOutput, err := execCmd.Output()

    if err != nil {
      return "", err
    }

    return string(cmdOutput), nil
  }

  OutputLocal(cmdStr)

  stdOutReader, err := execCmd.StdoutPipe()
  stdErrReader, err := execCmd.StderrPipe()

  if err != nil {
    return "", err
  }

  stdOutScanner := bufio.NewScanner(stdOutReader)
  go func() {
    for stdOutScanner.Scan() {
      OutputLocal(stdOutScanner.Text())
    }
  }()

  stdErrScanner := bufio.NewScanner(stdErrReader)
  go func() {
    for stdErrScanner.Scan() {
      OutputLocal(stdErrScanner.Text())
    }
  }()

  err = execCmd.Start()
  if err != nil {
    return "", err
  }

  err = execCmd.Wait()
  if err != nil {
    return "", err
  }

  return "", nil
}

func (config *ClientConfig) run(cmd string, useSudo bool) (string) {

  // create a wait group the waits for command execution on all hosts
  var sessionWaitGroup sync.WaitGroup
  sessionWaitGroup.Add(len(config.Hosts))

  // need to create a session for each host
  for _, host := range config.Hosts {
    client, session, err := CreateSession(&host)

    if err != nil {
      log.Fatalln(err)
      break
    }

    runner := runner {
      ClientConfig: config,
      Client: client,
      Session: session,
      Host: host,
      WaitGroup: &sessionWaitGroup,
      UseSudo: useSudo,
    }

    go runner.run(cmd)
  }

  sessionWaitGroup.Wait()

  return ""
}

func (runner *runner) run(cmd string) {

  defer runner.WaitGroup.Done()
  defer runner.Client.Close()
  defer runner.Session.Close()

  // Set up terminal modes
  modes := ssh.TerminalModes {
    ssh.ECHO:          0,     // disable echoing
    ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
    ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
  }

  if err := runner.Session.RequestPty("xterm", 80, 40, modes); err != nil {
    log.Fatalf("request for pseudo terminal failed: %s", err)
  }

  // switch the current working directory
  cdCmd := ""
  if runner.ClientConfig.CurrentDirectory != "" {
    cdCmd = "cd " + runner.ClientConfig.CurrentDirectory + " && "
  }

  cmdStr := cdCmd + cmd
  cmdDisplay := cmdStr

  sudoChannel := make(chan bool)
  if runner.UseSudo {
    cmdStr = fmt.Sprintf("/usr/bin/sudo bash <<CMD\nexport PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin:/root/bin\n%s\nCMD", cmdStr)
    sudoRunner := sudoRunner{}
    sudoRunner.handlePrompt(runner, sudoChannel)
  }

  OutputRemote(runner.Host, cmdDisplay)

  // conditionally create the stdOut reader. If using sudo, the sudo handler will
  // handle displaying output.
  if !runner.UseSudo {
    stdOutReader, err := runner.Session.StdoutPipe()
    runner.WaitGroup.Add(1)
    if err != nil {
      log.Fatalln(err)
    }

    stdOutScanner := bufio.NewScanner(stdOutReader)
    go func() {
      for stdOutScanner.Scan() {
        OutputRemote(runner.Host, stdOutScanner.Text())
      }

      defer runner.WaitGroup.Done()
    }()
  }

  stdErrReader, err := runner.Session.StderrPipe()
  runner.WaitGroup.Add(1)
  if err != nil {
    log.Fatalln(err)
  }

  stdErrScanner := bufio.NewScanner(stdErrReader)
  go func() {
    for stdErrScanner.Scan() {
      OutputRemote(runner.Host, stdErrScanner.Text())
    }

    defer runner.WaitGroup.Done()
  }()

  if err := runner.Session.Run(cmdStr); err != nil {
    close(sudoChannel)
    panic("Failed to run: " + err.Error())
  }

  close(sudoChannel)
}