package gomez

import (
  "bufio"
  "fmt"
  "log"
  "os"
  "io"
  "sync"

  "golang.org/x/crypto/ssh"
)

type runner struct {
  host         *Host
  clientConfig *ClientConfig
  client       *ssh.Client
  session      *ssh.Session
  stdin        io.WriteCloser
  stdout       io.Reader
  waitGroup    *sync.WaitGroup
  cmdOptions   CmdOptions
}

// run a command on the remote hosts
func (config *ClientConfig) Run(cmd string) (CmdResult) {
  return config.run(cmd, CmdOptions {})
}

func (config *ClientConfig) RunWithOpts(cmd string, options CmdOptions) (CmdResult) {
  return config.run(cmd, options)
}

func (config *ClientConfig) run(cmd string, options CmdOptions) (CmdResult) {

  cmdResult := CmdResult{}

  // create a wait group the waits for command execution on all hosts
  var sessionWaitGroup sync.WaitGroup
  sessionWaitGroup.Add(len(config.Hosts))

  // need to create a session for each host
  for _, host := range config.Hosts {
    client, session, err := CreateSession(host)

    if err != nil {
      log.Fatalln(err)
      break
    }

    runner := runner {
      clientConfig: config,
      client:       client,
      session:      session,
      host:         host,
      waitGroup:    &sessionWaitGroup,
      cmdOptions:   options,
    }

    go runner.run(cmd)
  }

  sessionWaitGroup.Wait()

  return cmdResult
}

func (runner *runner) run(cmd string) {

  defer runner.waitGroup.Done()
  defer runner.client.Close()
  defer runner.session.Close()

  // Set up terminal modes
  modes := ssh.TerminalModes{
    ssh.ECHO:          0,     // disable echoing
    ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
    ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
  }

  if err := runner.session.RequestPty("xterm", 80, 40, modes); err != nil {
    log.Fatalf("request for pseudo terminal failed: %s", err)
  }

  // switch the current working directory
  cdCmd := ""
  if runner.cmdOptions.WorkingDirectory != "" {
    cdCmd = "cd " + runner.cmdOptions.WorkingDirectory + " && "
  }

  cmdStr := cdCmd + cmd
  cmdDisplay := cmdStr

  sudoChannel := make(chan bool)
  if runner.cmdOptions.UseSudo {
    cmdStr = WrapSudoCommand(cmdStr)
    sudoRunner := sudoRunner{}
    sudoRunner.handlePrompt(runner, sudoChannel)
  }

  OutputRemote(runner.host, cmdDisplay)

  // conditionally create the stdOut reader. If using sudo, the sudo handler will
  // handle displaying output.
  if !runner.cmdOptions.UseSudo {
    stdOutReader, err := runner.session.StdoutPipe()
    runner.waitGroup.Add(1)
    if err != nil {
      log.Fatalln(err)
    }

    stdOutScanner := bufio.NewScanner(stdOutReader)
    go func() {
      for stdOutScanner.Scan() {
        OutputRemote(runner.host, stdOutScanner.Text())
      }

      defer runner.waitGroup.Done()
    }()
  }

  stdErrReader, err := runner.session.StderrPipe()
  runner.waitGroup.Add(1)
  if err != nil {
    log.Fatalln(err)
  }

  stdErrScanner := bufio.NewScanner(stdErrReader)
  go func() {
    for stdErrScanner.Scan() {
      OutputRemote(runner.host, stdErrScanner.Text())
    }

    defer runner.waitGroup.Done()
  }()

  if err := runner.session.Run(cmdStr); err != nil {
    close(sudoChannel)
    os.Exit(1)
  }

  close(sudoChannel)
}