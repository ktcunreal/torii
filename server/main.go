package main

import (
	"../config"
	"../proxy"
	"../utils"
	"log"
	"net"
)

func main() {
	conf := config.LoadServerConf()
	key := utils.NewKey(conf.RAW)

	socks := initAddr(conf.SOCKSSERVER)
	defer socks.Close()

	if len(conf.TCPSERVER)*len(conf.UPSTREAM) > 0 {
		tcp := initAddr(conf.TCPSERVER)
		defer tcp.Close()
		go func() {
			for {
				src, err := tcp.Accept()
				if err != nil {
					log.Printf("TCP SERVER FAILED TO ACCEPT CONNECTION: %v", err)
					continue
				}
				dst, err := net.Dial("tcp", conf.UPSTREAM)
				if err != nil {
					log.Println("UNABLE TO CONNECT UPSTREAM SERVER")
					continue
				}
				forward(src, dst, conf, key)
			}
		}()
	}

	for {
		client, err := socks.Accept()
		if err != nil {
			log.Printf("SOCKS SERVER FAILED TO ACCEPT CONNECTION: %v", err)
			continue
		}
		go socks5(client, conf, key)
	}
}

func initAddr(addr string) net.Listener {
	defer log.Printf("LISTENER STARTED ON %v", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("LISTENER FAILED TO START: ", err)
	}
	return listener
}

func socks5(client net.Conn, conf *config.Server, key *utils.Key) {
	eStream := utils.NewEncStream(client, key)
	switch conf.COMPRESSION {
	case "none":
		proxy.NewProxyServer(eStream).Connect()
	case "snappy":
		cStream := utils.NewSnappyStream(eStream)
		proxy.NewProxyServer(cStream).Connect()
	default:
		cStream := utils.NewSnappyStream(eStream)
		proxy.NewProxyServer(cStream).Connect()
	}
}

func forward(src, dst net.Conn, conf *config.Server, key *utils.Key) {
	eStream := utils.NewEncStream(src, key)
	switch conf.COMPRESSION {
	case "none":
		proxy.Pipe(eStream, dst)
	case "snappy":
		cStream := utils.NewSnappyStream(eStream)
		proxy.Pipe(cStream, dst)
	default:
		cStream := utils.NewSnappyStream(eStream)
		proxy.Pipe(cStream, dst)
	}
}
