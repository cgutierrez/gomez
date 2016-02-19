package gomez

import (
  "fmt"
  "io"
  "log"
  "os"
  "path/filepath"
  "sync"
)

func (config *ClientConfig) Put(source string, destination string) {

  files, err := filepath.Glob(source)
  if err != nil {
    log.Fatalln(err)
  }

  for _, file := range files {

    fileReader, err := os.Open(file)
    if err != nil {
      continue
    }

    fileInfo, err := os.Stat(file)
    if err != nil {
      continue
    }

    mode := uint32(fileInfo.Mode().Perm())
    header := fmt.Sprintf("C%04o %d %s\n", mode, fileInfo.Size(), filepath.Base(file))

    var sessionWaitGroup sync.WaitGroup

    for _, host := range config.Hosts {

      sessionWaitGroup.Add(1)
      go func() {

        _, session, err := CreateSession(host)
        if err != nil {
          log.Fatalln(err)
        }

        defer session.Close()
        defer sessionWaitGroup.Done()

        go func() {

          OutputLocal(fmt.Sprintf("copy %s (%dMB) to %s:%s", file, fileInfo.Size() / 1024 / 2014, host.Host, destination))

          stdinPipe, _ := session.StdinPipe()
          defer stdinPipe.Close()

          _, err = stdinPipe.Write([]byte(header))

          if err != nil {
            return
          }

          _, err = io.Copy(stdinPipe, fileReader)

          if err != nil {
            return
          }

          _, err = stdinPipe.Write([]byte{0})

          if err != nil {
            return
          }
        }();

        if err = session.Run("/usr/bin/scp -trv " + destination); err != nil {
          os.Exit(1)
        }
      }()

      sessionWaitGroup.Wait()
    }
  }
}