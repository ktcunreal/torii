package config

import (
	"github.com/ktcunreal/torii/utils"
	"encoding/json"
	"log"
	"flag"
	"os"
)

type Server struct {
	PSK    [32]byte
	RAW    string `json:"key"`
	SERVER string `json:"serveraddr"`
	COMPRESSION string `json:"compression"`
}


func LoadServer() *Server {
	path := flag.String("c", "./config.json", "CONFIG FILE PATH")
	flag.Parse()
	log.Printf("LOADING CONFIG FROM %v", *path)

	conf := &Server{}
	file, err := os.Open(*path)
	if err != nil {
		log.Fatalf("LOAD CONFIG ERROR: %v", err)
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(conf); err != nil {
		log.Fatalf("PARSE CONFIG ERROR: %v", err)
	}

	conf.PSK, conf.RAW = utils.SH256R(conf.RAW), ""
	utils.HKEY1 = utils.SH256L(conf.PSK[:10])
	utils.HKEY2 = utils.SH256L(conf.PSK[20:])
	return conf
}
