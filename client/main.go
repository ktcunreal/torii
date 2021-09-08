package main

import (
	"../config"
	"../proxy"
	"../utils"
	"log"
	"net"
)

func main() {
	conf := config.LoadClientConf()
	key := utils.NewKey(conf.RAW)

	socks := initAddr(conf.SOCKSCLIENT)
	defer socks.Close()

	if len(conf.TCPSERVER)*len(conf.TCPCLIENT) > 0 {
		tcp := initAddr(conf.TCPCLIENT)
		defer tcp.Close()
		go func() {
			for {
				src, err := tcp.Accept()
				if err != nil {
					log.Printf("TCP CLIENT FAILED TO ACCEPT CONNECTION: %v", err)
					continue
				}
				dst, err := net.Dial("tcp", conf.TCPSERVER)
				if err != nil {
					log.Println("UNABLE TO CONNECT TCP SERVER")
					continue
				}
				forward(src, dst, conf, key)
			}
		}()
	}

	for {
		client, err := socks.Accept()
		if err != nil {
			log.Println("FAILED TO ACCEPT CONNECTION: ", err)
			continue
		}
		server, err := net.Dial("tcp", conf.SOCKSSERVER)
		if err != nil {
			log.Println("COULD NOT CONNECT TO SERVER: ", err)
			continue
		}
		go socks5(server, client, conf, key)
	}
}

func initAddr(addr string) net.Listener {
	defer log.Printf("LISTENER STARTED AT %v", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("LISTENER FAILED TO START: ", err)
	}
	return listener
}

func socks5(server, client net.Conn, conf *config.Client, key *utils.Key) {
	eStream := utils.NewEncStream(server, key)
	switch conf.COMPRESSION {
	case "none":
		proxy.NewProxyClient(client).Connect(eStream)
	case "snappy":
		cStream := utils.NewSnappyStream(eStream)
		proxy.NewProxyClient(client).Connect(cStream)
	default:
		cStream := utils.NewSnappyStream(eStream)
		proxy.NewProxyClient(client).Connect(cStream)
	}
}

func forward(src, dst net.Conn, conf *config.Client, key *utils.Key) {
	eStream := utils.NewEncStream(dst, key)
	switch conf.COMPRESSION {
	case "none":
		proxy.Pipe(src, eStream)
	case "snappy":
		cStream := utils.NewSnappyStream(eStream)
		proxy.Pipe(src, cStream)
	default:
		cStream := utils.NewSnappyStream(eStream)
		proxy.Pipe(src, cStream)
	}
}
