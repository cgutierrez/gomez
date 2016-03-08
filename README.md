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

## Dependencies
```bash
go get github.com/gcmurphy/getpass
go get golang.org/x/crypto/ssh
```

## Extremely helpful libraries
- [https://github.com/wingedpig/loom](https://github.com/wingedpig/loom)
- [https://github.com/laher/scp-go](https://github.com/laher/scp-go)
