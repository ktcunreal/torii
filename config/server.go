package config

import (
	"encoding/json"
	"os"
)

type Server struct {
	PSK    [32]byte
	RAW    string `json:"key"`
	SERVER string `json:"serveraddr"`
	COMPRESSION string `json:"compression"`
}

func InitServer(path string) (*Server, error) {
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