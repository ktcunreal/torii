package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

type Client struct {
	SOCKSSERVER string `json:"socksserver"`
	SOCKSCLIENT string `json:"socksclient"`
	COMPRESSION string `json:"compression"`
	TCPSERVER   string `json:"tcpserver"`
	TCPCLIENT   string `json:"tcpclient"`
	RAW         string `json:"key"`
}

func LoadClientConf() *Client {
	socksserver := flag.String("s", "", "Socks server address")
	socksclient := flag.String("l", "", "Socks client address")
	tcpserver := flag.String("t", "", "Tcp server address")
	tcpclient := flag.String("b", "", "Tcp client address")
	config := flag.String("c", "./config.json", "Configuration path")
	comp := flag.String("z", "snappy", "Use compression")
	psk := flag.String("p", "", "Pre-shared Key")
	
	flag.Parse()

	file, err := os.Open(*config)
	defer file.Close()
	client := &Client{}

	switch err {
	case nil:
		log.Printf("LOADING CONFIG FROM %v", *config)
		if err := json.NewDecoder(file).Decode(client); err != nil {
			log.Fatalf("PARSE CONFIG ERROR: %v", err)
		}
	default:
		log.Println("PARSING CMDLINE ARGS")
		client.SOCKSSERVER = *socksserver
		client.SOCKSCLIENT = *socksclient
		client.TCPSERVER = *tcpserver
		client.TCPCLIENT = *tcpclient
		client.COMPRESSION = *comp
		client.RAW = *psk
	}

	if len(client.SOCKSCLIENT) > 0 && !validateIP(client.SOCKSCLIENT) {
		log.Fatalln("INVALID SOCKS CLIENT IP ADDRESS")
	}
	if len(client.SOCKSSERVER) > 0 && !validateIP(client.SOCKSSERVER) {
		log.Fatalln("INVALID SOCKS SERVER IP ADDRESS")
	}
	if len(client.TCPSERVER) > 0 && !validateIP(client.TCPSERVER) {
		log.Fatalln("INVALID TCP SERVER ADDRESS")
	}

	return client
}
