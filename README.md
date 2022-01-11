# Torii
A simple, easy-to-use tunnel utility written in go.

## Feature
- AEAD Cipher
- Lightweight
- Obfuscate message length
- Suspend illegal connections to filter active probing
- Stream compression for better web-browsing experience

## Download
Download binary from [github release page](https://github.com/ktcunreal/torii/releases)

## Build from source
*on Linux / Windows x86_64, Go 1.13.3 or newer*

```
git clone github.com/ktcunreal/torii
``` 
Install Dependencies:
```
go get -u golang.org/x/crypto/nacl/secretbox 
go get -u github.com/golang/snappy
```

Build binaries:
```
go build -o server server/main.go 
go build -o client client/main.go
```

## Usage

### Server side

#### Example

`./server -s "0.0.0.0:1234" -t "0.0.0.0:2345" -u "127.0.0.1:8123" -p "some-long-random-passphrase" -z "snappy"`

or

`./server -c /path/to/config.json`

```
{
    "socksserver": "0.0.0.0:1234",
    "tcpserver": "0.0.0.0:2345",
    "upstream": "127.0.0.1:8123",
    "compression": "snappy",
    "key": "some-long-random-passphrase"
}
```

### Client side

`./client -s "127.0.0.1:1234" -l "0.0.0.0:1080" -t "127.0.0.1:2345" -a "0.0.0.0:1081" -p "some-long-random-passphrase" -z "snappy"`

or

`./client -c /path/to/config.json`

```
{
    "socksserver": "127.0.0.1:1234",
    "socksclient": "0.0.0.0:1080",
    "tcpserver": "127.0.0.1:2345",
    "tcpclient": "0.0.0.0:1081",
    "compression": "snappy",
    "key": "some-long-random-passphrase"
}
```

*Use a password consist of alphanumeric and symbol, at least 20 digits in length (Recommended)*

## Reference

https://github.com/golang/snappy

https://golang.org/x/crypto/nacl/secretbox

https://github.com/xtaci/kcptun

https://gfw.report/blog/ss_advise/en/

https://gist.github.com/clowwindy/5947691

## License
[GNU General Public License v3.0](https://raw.githubusercontent.com/ktcunreal/torii/master/LICENSE)
