package config

import (
	"../encrypt"
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type Server struct {
	Socksserver string `json:"socksserver"`
	Compression string `json:"compression"`
	Tcpserver   string `json:"tcpserver"`
	Upstream    string `json:"upstream"`
	Psk         string `json:"key"`
	keyring     *encrypt.Keyring
}

func LoadServerConf() *Server {
	socksserver := flag.String("s", "", "Socks server address")
	tcpserver := flag.String("t", "", "Tcp server address")
	upstream := flag.String("u", "", "Upstream address")
	config := flag.String("c", "", "Configuration path")
	comp := flag.String("z", "", "Use compression")
	Psk := flag.String("p", "", "Pre-shared Keyring")
	flag.Parse()

	server := &Server{}
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
		server.Socksserver = *socksserver
	}
	if *tcpserver != "" {
		server.Tcpserver = *tcpserver
	}
	if *upstream != "" {
		server.Upstream = *upstream
	}
	if *comp != "" {
		server.Compression = *comp
	}
	if *Psk != "" {
		server.Psk = *Psk
	}
	if len(server.Socksserver) == 0 && len(server.Tcpserver)*len(server.Upstream) == 0 {
		log.Fatalln("INVALID ARGS FOR LISTENING ADDRESS")
	}
	if len(server.Socksserver) > 0 && !validateIP(server.Socksserver) {
		log.Fatalln("INVALID SOCKS SERVER ADDRESS")
	}
	if len(server.Tcpserver) > 0 && !validateIP(server.Tcpserver) {
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

func (s *Server) Getkeyring() *encrypt.Keyring {
	if s.keyring == nil {
		s.keyring = encrypt.NewKeyring(s.Psk)
	}
	return s.keyring
}
