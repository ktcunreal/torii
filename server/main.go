package main

import (
	"../config"
	"../proxy"
	"../utils"
	"log"
	"net"
)

func main() {
	conf := config.LoadServer()
	key := utils.NewKey(conf.RAW)
	server := initListener(conf.SERVER)
	defer server.Close()

	for {
		client, err := server.Accept()
		if err != nil {
			log.Printf("FAILED TO ACCEPT CONNECTION FROM CLIENT: %v", err)
			continue
		}
		go connect(client, conf, key)
	}
}

func initListener(addr string) net.Listener {
	defer log.Printf("LISTENER STARTED AT %v", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("LISTENER FAILED TO START: ", err)
	}
	return listener
}

func connect(client net.Conn, conf *config.Server, key *utils.Key) {
	eStream := utils.NewEncStream(client, key)
	switch conf.COMPRESSION {
	case "none":
		proxy.NewProxyServer(eStream).Forward()
	case "snappy":
		cStream := utils.NewSnappyStream(eStream)
		proxy.NewProxyServer(cStream).Forward()
	default:
		cStream := utils.NewSnappyStream(eStream)
		proxy.NewProxyServer(cStream).Forward()
	}
}
