package config

import (
	// "../utils"
	"encoding/json"
	"flag"
	"os"
	"log"
)

type Client struct {
	RAW    string `json:"key"`
	SERVER string `json:"serveraddr"`
	CLIENT string `json:"clientaddr"`
	COMPRESSION string `json:"compression"`
}

func LoadClient() *Client {
	config := &Client{}
	c := flag.String("c", "./config.json", "Configuration path")
	p := flag.String("p","","Pre-shared Key")
	s := flag.String("s","127.0.0.1:5234","Server address")
	l := flag.String("l","0.0.0.0:8907","Client address")
	z := flag.String("z","snappy","Use compression")

	flag.Parse()
	log.Printf("LOADING CONFIG FROM %v", *c)

	file, err := os.Open(*c)	
	if err != nil {
		log.Printf("COULD NOT LOAD CONFIG: %v, TRYING TO PARSE CMDLINE ARGS", err)
		config.SERVER = *s
		config.CLIENT = *l
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

