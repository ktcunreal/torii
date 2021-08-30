package config

import (
	"github.com/ktcunreal/torii/utils"
	"encoding/json"
	"flag"
	"os"
	"log"
)

type Client struct {
	PSK    [32]byte
	RAW    string `json:"key"`
	SERVER string `json:"serveraddr"`
	CLIENT string `json:"clientaddr"`
	COMPRESSION string `json:"compression"`

}

func LoadClient() *Client {
	path := flag.String("c", "./config.json", "CONFIGURATION PATH")
	flag.Parse()
	log.Printf("LOADING CONFIG FROM %v", *path)

	conf := &Client{}
	file, err := os.Open(*path)
	if err != nil {
		log.Fatalf("LOAD CONFIG ERROR: %v", err)
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(conf)
	conf.PSK, conf.RAW = utils.SH256R(conf.RAW), ""

	utils.HKEY1 = utils.SH256L(conf.PSK[:10])
	utils.HKEY2 = utils.SH256L(conf.PSK[20:])
	return conf
}

