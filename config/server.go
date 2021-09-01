package config

import (
	"encoding/json"
	"log"
	"flag"
	"os"
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
	s := flag.String("s","0.0.0.0:8907","Server address")
	z := flag.String("z","snappy","Use compression")

	flag.Parse()
	log.Printf("LOADING CONFIG FROM %v", *c)

	file, err := os.Open(*c)
	if err != nil {
		log.Printf("COULD NOT LOAD CONFIG: %v, TRYING TO PARSE CMDLINE ARGS", err)
		config.SERVER = *s
		config.COMPRESSION = *z
		config.RAW = *p
		return config
	}
	defer file.Close()
	
	if err := json.NewDecoder(file).Decode(config); err != nil {
		log.Fatalf("PARSE CONFIG ERROR: %v", err)
	}

	return config
}
