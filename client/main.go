package main

import (
	"github.com/ktcunreal/torii/config"
	"github.com/ktcunreal/torii/utils"
	"github.com/ktcunreal/torii/proxy"
	"log"
	"net"
)

func main() {
	conf := config.LoadClient()
	listener := initLsnr(conf.CLIENT)
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
		go initConn(server, client, conf)
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

func initConn(server, client net.Conn, conf *config.Client) {
	eStream := utils.NewEncStream(server, &conf.PSK)
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
