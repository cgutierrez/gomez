package gomez

import (
  "bufio"
  "fmt"
  _"log"
  "os"
  "os/exec"
  "sync"
)

// run a local command
func (config *ClientConfig) Local(cmd string) (CmdResult) {
  return config.local(cmd, CmdOptions {})
}

func (config *ClientConfig) LocalWithOpts(cmd string, options CmdOptions) (CmdResult) {
  return config.local(cmd, options)
}

func (config *ClientConfig) local(cmd string, options CmdOptions) (CmdResult) {

  cmdResult := CmdResult {}

  var sessionWaitGroup sync.WaitGroup
  sessionWaitGroup.Add(2)

  cdCmd := ""
  if options.WorkingDirectory != "" {
    cdCmd = "cd " + options.WorkingDirectory + " && "
  }

  cmdStr := cdCmd + cmd

  if options.UseSudo {
    cmdStr = WrapSudoCommand(cmdStr)
  }

  execCmd := exec.Command("/bin/sh", "-c", cmdStr)

  stdOutReader, err := execCmd.StdoutPipe()

  if err != nil {
    cmdResult.Error = err
    return cmdResult
  }

  stdOutScanner := bufio.NewScanner(stdOutReader)

  go func() {
    defer sessionWaitGroup.Done()
    for stdOutScanner.Scan() {
      if options.CaptureOutput {
        cmdResult.Result = fmt.Sprintln(cmdResult.Result + stdOutScanner.Text())
        continue
      }

      OutputLocal(stdOutScanner.Text())
    }
  }()

  stdErrReader, err := execCmd.StderrPipe()

  if err != nil {
    cmdResult.Error = err
    return cmdResult
  }

  stdErrScanner := bufio.NewScanner(stdErrReader)
  go func() {
    defer sessionWaitGroup.Done()
    for stdErrScanner.Scan() {
      OutputLocal(stdErrScanner.Text())
    }
  }()

  err = execCmd.Start()
  if err != nil {
    os.Exit(1)
  }

  sessionWaitGroup.Wait()
  execCmd.Wait()

  return cmdResult
}