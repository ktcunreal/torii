# Torii
A simple, easy-to-use tunnel utility written in go.

## Feature
- Use crypto from trusted source (NaCl in Official Go cryptography library)
- Lightweight protocol
- The length of encrypted messages is masked
- Enable nonce to filter replay attack
- Drop illegal connections to protect server against fingerprint probing
- Support appropriate stream compression for web browsing

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
go get -u github.com/klauspost/compress/s2
```

Build binaries:
```
go build -o server server/main.go 
go build -o client client/main.go
```

## Usage
Torii reads config.json in current working directory by default
### server config

```
{
    "serveraddr": "0.0.0.0:8462",
    "compression": "snappy",
    "key": "some-long-random-passphrase"
}
```
or specify the location of your config file 
> ./server -c /path/to/your/config/file 

### client config

```
{
    "serveraddr": "127.0.0.1:8642",
    "clientaddr": "0.0.0.0:1234",
    "compression": "snappy",
    "key": "some-long-random-passphrase"
}
```

> ./client -c /path/to/your/config/file 

*Use a password consist of alphanumeric and symbol, at least 20 digits in length (Recommended)*

## Reference
https://github.com/klauspost/compress

https://github.com/golang/snappy

https://golang.org/x/crypto/nacl/secretbox

https://github.com/xtaci/kcptun

https://gfw.report/blog/ss_advise/en/

https://gist.github.com/clowwindy/5947691

## License
[GNU General Public License v3.0](https://raw.githubusercontent.com/ktcunreal/torii/master/LICENSE)
