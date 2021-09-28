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
	RAW         string `json:"key"`
}

func LoadServerConf() *Server {
	socksserver := flag.String("s", "", "Socks server address")
	tcpserver := flag.String("t", "", "Tcp server address")
	upstream := flag.String("u", "", "Upstream address")
	config := flag.String("c", "./config.json", "Configuration path")
	comp := flag.String("z", "snappy", "Use compression")
	psk := flag.String("p", "", "Pre-shared Key")

	flag.Parse()

	file, err := os.Open(*config)
	defer file.Close()
	server := &Server{}

	switch err {
	case nil:
		log.Printf("LOADING CONFIG FROM %v", *config)
		if err := json.NewDecoder(file).Decode(server); err != nil {
			log.Fatalf("PARSE CONFIG ERROR: %v", err)
		}
	default:
		log.Println("PARSING CMDLINE ARGS")
		server.SOCKSSERVER = *socksserver
		server.TCPSERVER = *tcpserver
		server.UPSTREAM = *upstream
		server.COMPRESSION = *comp
		server.RAW = *psk
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
