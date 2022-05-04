package config

import (
	"../encrypt"
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Client struct {
	Socksserver string `json:"socksserver"`
	Socksclient string `json:"socksclient"`
	Compression string `json:"compression"`
	Tcpserver   string `json:"tcpserver"`
	Tcpclient   string `json:"tcpclient"`
	Psk         string `json:"key"`
	keyring     *encrypt.Keyring
}

func LoadClientConf() *Client {
	socksserver := flag.String("s", "", "Socks server address")
	socksclient := flag.String("l", "", "Socks client address")
	tcpserver := flag.String("t", "", "Tcp server address")
	tcpclient := flag.String("a", "", "Tcp client address")
	config := flag.String("c", "", "Configuration path")
	comp := flag.String("z", "", "Use compression")
	Psk := flag.String("p", "", "Pre-shared Keyring")
	flag.Parse()

	client := &Client{}
	if *config != "" {
		file, err := os.Open(*config)
		defer file.Close()
		log.Printf("LOADING CONFIG FROM %v", *config)
		if err != nil {
			log.Fatalf("LOADING CONFIG FAILED %v", err)
		}
		if err := json.NewDecoder(file).Decode(client); err != nil {
			log.Fatalf("PARSE CONFIG ERROR: %v", err)
		}
	}

	if *socksserver != "" {
		client.Socksserver = *socksserver
	}
	if *socksclient != "" {
		client.Socksclient = *socksclient
	}
	if *tcpserver != "" {
		client.Tcpserver = *tcpserver
	}
	if *tcpclient != "" {
		client.Tcpclient = *tcpclient
	}
	if *comp != "" {
		client.Compression = *comp
	}
	if *Psk != "" {
		client.Psk = *Psk
	}

	if len(client.Socksserver)*len(client.Socksclient) == 0 && len(client.Tcpserver)*len(client.Tcpclient) == 0 {
		log.Fatalln("INVALID ARGS FOR LISTENING ADDRESS")
	}
	if len(client.Socksclient) > 0 && !validateIP(client.Socksclient) {
		log.Fatalln("INVALID SOCKS CLIENT IP ADDRESS")
	}
	if len(client.Tcpclient) > 0 && !validateIP(client.Tcpclient) {
		log.Fatalln("INVALID TCP CLIENT ADDRESS")
	}
	if len(client.Socksserver) > 0 {
		if !validateIP(client.Socksserver) && !validateDomain(client.Socksserver) {
			log.Fatalln("INVALID SOCKS SERVER ADDRESS")
		}
	}
	if len(client.Tcpserver) > 0 {
		if !validateIP(client.Tcpserver) && !validateDomain(client.Tcpserver) {
			log.Fatalln("INVALID TCP SERVER ADDRESS")
		}
	}

	return client
}

func validateIP(s string) bool {
	var ip, port string
	i := strings.LastIndexByte(s, ':')
	if i == -1 {
		return false
	}
	ip, port = s[:i], s[i+1:]
	if len(ip) == 0 || net.ParseIP(ip) == nil {
		return false
	}
	if p, err := strconv.Atoi(port); err != nil || p > 65535 || p < 1 {
		return false
	}
	return true
}

func validateDomain(s string) bool {
	RegExp := regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z
 ]{2,3})$`)
	if RegExp.MatchString(s) {
		return true
	}
	return false
}

func (c *Client) Getkeyring() *encrypt.Keyring {
	if c.keyring == nil {
		c.keyring = encrypt.NewKeyring(c.Psk)
	}
	return c.keyring
}
