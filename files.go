package gomez

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/ssh"
)

func writeFileToHost(session *ssh.Session, host *Host, file string, fileReader *os.File, fileInfo os.FileInfo, destination string, sessionWaitGroup *sync.WaitGroup) {

	defer sessionWaitGroup.Done()
	defer session.Close()

	sessionWaitGroup.Add(1)
	go func() {
		defer sessionWaitGroup.Done()

		mode := uint32(fileInfo.Mode().Perm())
		header := fmt.Sprintf("C%04o %d %s\n", mode, fileInfo.Size(), filepath.Base(file))

		OutputLocal(fmt.Sprintf("copy %s (%dMB) to %s:%s", file, fileInfo.Size()/1024/2014, host.Host, destination))

		stdinPipe, _ := session.StdinPipe()
		defer stdinPipe.Close()

		_, err := stdinPipe.Write([]byte(header))

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

	}()

	if err := session.Run("/usr/bin/scp -trv " + destination); err != nil {
		fmt.Println("in here")
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func (config *ClientConfig) Put(source string, destination string) {

	files, err := filepath.Glob(source)
	if err != nil {
		log.Fatalln(err)
	}

	var sessionWaitGroup sync.WaitGroup

	for _, file := range files {

		// upload each file to each host
		for _, host := range config.Hosts {

			_, session, err := CreateSession(host)
			if err != nil {
				log.Fatalln(err)
			}

			fileReader, err := os.Open(file)
			if err != nil {
				continue
			}

			fileInfo, err := os.Stat(file)
			if err != nil {
				continue
			}

			sessionWaitGroup.Add(1)
			go writeFileToHost(session, host, file, fileReader, fileInfo, destination, &sessionWaitGroup)
		}

	}

	sessionWaitGroup.Wait()
}
