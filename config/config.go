package config

import (
	"crypto/sha256"
	"encoding/json"
	"os"
)

type Server struct {
	PSK    [32]byte
	RAW    string `json:"key"`
	SERVER string `json:"serveraddr"`
}

type Client struct {
	PSK    [32]byte
	RAW    string `json:"key"`
	SERVER string `json:"serveraddr"`
	CLIENT string `json:"clientaddr"`
}

func LoadSC(path string) (*Server, error) {
	server := &Server{}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(server)
	server.PSK, server.RAW = SH256(server.RAW), ""
	return server, err
}

func LoadCC(path string) (*Client, error) {

	client := &Client{}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(client)
	client.PSK, client.RAW = SH256(client.RAW), ""
	return client, err
}

func SH256(s string) [32]byte {
	return sha256.Sum256([]byte(s))
}
