package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

type Client struct {
	RAW         string `json:"key"`
	SERVER      string `json:"serveraddr"`
	CLIENT      string `json:"clientaddr"`
	COMPRESSION string `json:"compression"`
}

func LoadClient() *Client {
	config := &Client{}
	c := flag.String("c", "./config.json", "Configuration path")
	p := flag.String("p", "", "Pre-shared Key")
	s := flag.String("s", "127.0.0.1:8000", "Server address")
	l := flag.String("l", "0.0.0.0:9000", "Client address")
	z := flag.String("z", "snappy", "Use compression")
	flag.Parse()

	log.Printf("LOADING CONFIG FROM %v", *c)
	file, err := os.Open(*c)
	defer file.Close()

	switch err {
	case nil:
		if err := json.NewDecoder(file).Decode(config); err != nil {
			log.Fatalf("PARSE CONFIG ERROR: %v", err)
		}
		if !validate(config.SERVER) || !validate(config.CLIENT) {
			log.Fatalln("Invalid IP ADDRESS")
		}
	default:
		log.Println("COULD NOT READ CONFIG FROM FILE, PARSING CMDLINE ARGS")
		config.RAW = *p
		config.COMPRESSION = *z
		config.SERVER = *s
		config.CLIENT = *l
		if !validate(config.SERVER) || !validate(config.CLIENT) {
			log.Fatalln("INVALID IP ADDRESS")
		}
	}
	return config
}
