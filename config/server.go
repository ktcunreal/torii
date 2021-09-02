package config

import (
	"encoding/json"
	"log"
	"flag"
	"net"
	"strings"
	"os"
	"strconv"
)

type Server struct {
	RAW    string `json:"key"`
	SERVER string `json:"serveraddr"`
	COMPRESSION string `json:"compression"`
}

func LoadServer() *Server {
	config := &Server{}
	c := flag.String("c", "./config.json", "Configuration path")
	p := flag.String("p","","Pre-shared Key")
	s := flag.String("s","0.0.0.0:8000","Server address")
	z := flag.String("z","snappy","Use compression")
	flag.Parse()

	file, err := os.Open(*c)
	log.Printf("LOADING CONFIG FROM %v", *c)
	defer file.Close()

	switch err {
		case nil:
			if err := json.NewDecoder(file).Decode(config); err != nil {
				log.Fatalf("PARSE CONFIG ERROR: %v", err)
			}
			if !validate(config.SERVER) { 
				log.Fatalln("Invalid IP ADDRESS")
			}
		default:
			log.Println("COULD NOT READ CONFIG FROM FILE, PARSING CMDLINE ARGS")
			config.RAW = *p
			config.COMPRESSION = *z
			config.SERVER = *s
			if !validate(config.SERVER) { 
				log.Fatalln("INVALID IP ADDRESS")
			}
	}
	return config
}

func validate(s string) bool {
	var ip, port string
	i := strings.LastIndexByte(s, ':')
	if i == -1 {
		return false
	}
	ip, port = s[:i], s[i+1:]
	if len(ip) == 0 || net.ParseIP(ip) == nil{
		return false
	}
	if p, err := strconv.Atoi(port); err != nil || p > 65535 || p < 1 {
		return false
	}
	return true
}