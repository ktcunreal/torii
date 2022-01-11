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
	PSK         string `json:"key"`
}

func LoadClientConf() *Client {
	client := &Client{}

	socksserver := flag.String("s", "", "Socks server address")
	socksclient := flag.String("l", "", "Socks client address")
	tcpserver := flag.String("t", "", "Tcp server address")
	tcpclient := flag.String("a", "", "Tcp client address")
	config := flag.String("c", "", "Configuration path")
	comp := flag.String("z", "snappy", "Use compression")
	psk := flag.String("p", "", "Pre-shared Key")
	
	flag.Parse()

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
		client.SOCKSSERVER = *socksserver
	}
	if *socksclient != "" {
		client.SOCKSCLIENT = *socksclient
	}
	if *tcpserver != "" {
		client.TCPSERVER = *tcpserver
	}
	if *tcpclient != "" {
		client.TCPCLIENT = *tcpclient
	}
	if *comp != "" {
		client.COMPRESSION = *comp
	}
	if *psk != "" {
		client.PSK = *psk
	}
	if len(client.SOCKSSERVER)*len(client.SOCKSCLIENT) == 0 && len(client.TCPSERVER)*len(client.TCPCLIENT) == 0 {
		log.Fatalln("INVALID ARGS FOR LISTENING ADDRESS")
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
	if len(client.TCPCLIENT) > 0 && !validateIP(client.TCPCLIENT) {
		log.Fatalln("INVALID TCP CLIENT ADDRESS")
	}
	return client
}
