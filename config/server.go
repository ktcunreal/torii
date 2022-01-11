package config

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type Server struct {
	SOCKSSERVER string `json:"socksserver"`
	COMPRESSION string `json:"compression"`
	TCPSERVER   string `json:"tcpserver"`
	UPSTREAM    string `json:"upstream"`
	PSK         string `json:"key"`
}

func LoadServerConf() *Server {
	server := &Server{}

	socksserver := flag.String("s", "", "Socks server address")
	tcpserver := flag.String("t", "", "Tcp server address")
	upstream := flag.String("u", "", "Upstream address")
	config := flag.String("c", "", "Configuration path")
	comp := flag.String("z", "", "Use compression")
	psk := flag.String("p", "", "Pre-shared Key")

	flag.Parse()

	if *config != "" {
		file, err := os.Open(*config)
		defer file.Close()
		log.Printf("LOADING CONFIG FROM %s", *config)
		if err != nil {
			log.Fatalf("LOADING CONFIG FAILED %v", err)
		}
		if err := json.NewDecoder(file).Decode(server); err != nil {
			log.Fatalf("PARSE CONFIG ERROR: %v", err)
		}
	}
	if *socksserver != "" {
		server.SOCKSSERVER = *socksserver
	}
	if *tcpserver != "" {
		server.TCPSERVER = *tcpserver
	}
	if *upstream != "" {
		server.UPSTREAM = *upstream
	}
	if *comp != "" {
		server.COMPRESSION = *comp
	}
	if *psk != "" {
		server.PSK = *psk
	}
	if len(server.SOCKSSERVER) == 0 && len(server.TCPSERVER)*len(server.UPSTREAM) == 0 {
		log.Fatalln("INVALID ARGS FOR LISTENING ADDRESS")
	}
	if len(server.SOCKSSERVER) > 0 && !validateIP(server.SOCKSSERVER) {
		log.Fatalln("INVALID SOCKS SERVER ADDRESS")
	}
	if len(server.TCPSERVER) > 0 && !validateIP(server.TCPSERVER) {
		log.Fatalln("INVALID TCP SERVER ADDRESS")
	}
	return server
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
