package gomez

import (
	"bufio"
	"fmt"
	"io"
	"log"

	"github.com/gcmurphy/getpass"
)

type sudoRunner struct {
}

type sudoMatcher struct {
	currentIndexMatch int
	currentPrompt     string
	stringToFind      string
	totalMatchLength  int
}

func WrapSudoCommand(cmd string) (string) {
  return fmt.Sprintf("/usr/bin/sudo bash <<CMD\nexport PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin:/root/bin\n%s\nCMD", cmd)
}

func (sudo *sudoRunner) handlePrompt(runner *runner, done <-chan bool) {

	writer, err := runner.session.StdinPipe()
	if err != nil {
		panic("Failed to run: " + err.Error())
	}

	reader, err := runner.session.StdoutPipe()
	if err != nil {
		panic("Failed to run: " + err.Error())
	}

	go sudo.sendPassword(runner, writer, reader, done)
}

func (sudo *sudoRunner) promptForPassword(runner *runner, writer io.Writer) (string, error) {

	if runner.host.Password == "" {

		password, err := getpass.GetPassWithOptions(fmt.Sprintf("enter sudo password for %s: ", runner.host.Host), 0, 100)

		if err != nil {
			return password, err
		}

		runner.host.Password = password
	}

	return runner.host.Password, nil
}

func (sudo *sudoRunner) sendPassword(runner *runner, writer io.Writer, reader io.Reader, done <-chan bool) {

	matcher := newSudoMatcher(runner.host.User)

	for {
		select {
		case <-done:
		default:
			bytesRead := make([]byte, matcher.totalMatchLength)
			_, err := reader.Read(bytesRead)

			if err == io.EOF {
				continue
			}

			if err != nil {
				log.Fatalln(err)
				return
			}

			matchFound := matcher.Match(bytesRead)
			if matchFound {

				password, err := sudo.promptForPassword(runner, writer)

				if err != nil {
					log.Fatalln(err)
				}

				writer.Write([]byte(password + "\n"))
				break
			}

			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {

				if scanner.Text() == "Sorry, try again." {

					runner.host.Password = ""
					password, err := sudo.promptForPassword(runner, writer)

					if err != nil {
						log.Fatalln(err)
					}

					writer.Write([]byte(password + "\n"))
					continue
				}

				if matcher.Match(scanner.Bytes()) {
					continue
				}

				OutputRemote(runner.host, scanner.Text())
			}

			break
		}
	}
}

func newSudoMatcher(user string) sudoMatcher {
	stringToFind := fmt.Sprintf("[sudo] password for %s:", user)
	totalMatchLength := len([]byte(stringToFind))
	return sudoMatcher{0, "", stringToFind, totalMatchLength}
}

func (matcher *sudoMatcher) Match(additionalBytes []byte) bool {
	readString := string(additionalBytes)

	for _, runeVal := range readString {
		if len(matcher.stringToFind) > matcher.currentIndexMatch && runeVal == rune(matcher.stringToFind[matcher.currentIndexMatch]) {
			matcher.currentPrompt = matcher.currentPrompt + string(runeVal)
			matcher.currentIndexMatch++
		} else {
			matcher.currentPrompt = ""
			matcher.currentIndexMatch = 0
		}

		if len(matcher.currentPrompt) == matcher.totalMatchLength {
			return true
		}
	}
	return false
}
