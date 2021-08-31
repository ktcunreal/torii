package main

import (
	"../config"
	"../utils"
	"../proxy"
	"log"
	"net"
)

func main() {
	conf := config.LoadClient()
	key := utils.NewKey(conf.RAW)

	listener := initListener(conf.CLIENT)
	defer listener.Close()

	for {
		client, err := listener.Accept()
		if err != nil {
			log.Println("FAILED TO ACCEPT CONNECTION: ", err)
			continue
		}
		server, err := net.Dial("tcp", conf.SERVER)
		if err != nil {
			log.Println("COULD NOT CONNECT TO SERVER: ", err)
			continue
		}
		go connect(server, client, conf, key)
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

func connect(server, client net.Conn, conf *config.Client, key *utils.Key) {
	eStream := utils.NewEncStream(server, key)
	switch conf.COMPRESSION {
		case "none":
			proxy.NewProxyClient(client).Forward(eStream)
		case "snappy":
			cStream := utils.NewSnappyStream(eStream)
			proxy.NewProxyClient(client).Forward(cStream)
		default:
			cStream := utils.NewSnappyStream(eStream)
			proxy.NewProxyClient(client).Forward(cStream)
	}
}
