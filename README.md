## Credit where it's due
The following libraries helped a ton while building this one.
- [https://github.com/wingedpig/loom](https://github.com/wingedpig/loom) - Loom is actually a very similar library to this one but operates on a single host (at least at the time of writing this one).
- [https://github.com/laher/scp-go](https://github.com/laher/scp-go)

## Dependencies
```bash
go get github.com/gcmurphy/getpass
go get github.com/cgutierrez/goshen
go get golang.org/x/crypto/ssh
```

## Creating hosts for Gomez clients
The `CreateHosts` method accepts a slice of string maps (`[]map[string]string`) that contain connection information for each host.
Valid key names in each map are:

- host
- user
- port
- keyFile

Gomez will take the value for `host` to search for connection information in your ssh config.

```go

host := map[string]string{"user": "foo", "host": "localhost", "port": "2200"}
hosts := make([]map[string]string, 0)
hosts = append(hosts, host)

gomez.CreateHosts(hosts)
```

## Creating a new client
There are two ways to create a client. To work with both local and remote SSH calls, use the `NewClient` method.

```go
// create a client with one or more hosts
client := gomez.NewClient(hosts)
```

In some instances, it can be useful to create a client that doesn't require hosts if you're just doing local execution.

```go
// create a client with one or more hosts
client := gomez.NewLocalClient()
```

## Running commands
There are four main method for running commands. `Run`, `RunWithOpts`, `Local`, and `LocalWithOpts`. The `Run` methods execute commands
on remote hosts while the `Local` methods run commands on the local host.

```go
// create a client with one or more hosts
client := gomez.NewClient(hosts)

// run the ls -l command on the remote host
client.Run("ls -l")

// run ls -l on the local host
client.Local("ls -l")
```

The `*WithOpts` methods take two arguments. The first is the command to run. The second is an instance of the `gomez.CmdOptions` struct.

```go
// create a client with one or more hosts
client := gomez.NewClient(hosts)

// run the ls -l command on the remote host using sudo in the /var/www directory
client.RunWithOpts("ls -l", gomez.CmdOptions { UseSudo: true, WorkingDirectory: "/var/www" })

// run the ls -l command on the local host using sudo in the /var/www directory
client.Local("ls -l", gomez.CmdOptions { UseSudo: true, CaptureOutput: true, WorkingDirectory: "/var/www" })
```

`gomez.CmdOptions` includes the following fields:

- UseSudo (bool - run the command using sudo)
- WorkingDirectory (string - change the working directory before running the command)
- CaptureOutput (bool - local only - returns the output of the command)

## Tests
Where the f*** are the tests? They're coming... the goal is to include a Go SSH server that can be used for testing. 
Currently, all the tests a private and depending on certain SSH keys existing.

## Build on OS X
If you're getting an error like `fatal error: 'openssl/ui.h' file not found`, it's coming from https://github.com/gcmurphy/getpass which uses cgo.
In order to build this correctly on OS X, You need to install OpenSSL using homebrew
```bash
brew install openssl && brew link openssl --force
```
Then you need to create a symlink to OpenSSL in `/Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/include` with

```bash
cd /Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/include
sudo ln -s /usr/local/opt/openssl/include/openssl openssl
```
