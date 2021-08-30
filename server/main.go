package main

import (
	"../config"
	"../utils"
	"../proxy"
	"log"
	"net"
)

func main() {
	conf := config.LoadServer()
	server := initLsnr(conf.SERVER)
	defer server.Close()

	for {
		client, err := server.Accept()
		if err != nil {
			log.Printf("FAILED TO ACCEPT CONNECTION FROM CLIENT: %v", err)
			continue
		}
		go initConn(client, conf)
	}
}

func initLsnr(addr string) net.Listener {
	defer log.Printf("LISTENER STARTED AT %v", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("LISTENER FAILED TO START: ", err)
	}
	return listener
}

func initConn(client net.Conn, conf *config.Server) {
	eStream := utils.NewEncStream(client, &conf.PSK)
	switch conf.COMPRESSION{
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
