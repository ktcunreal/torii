package main

import (
	"../config"
	"../utils"
	"../proxy"
	"flag"
	"log"
	"net"
)

func main() {
	conf := initConf()
	server := initLsnr(conf.SERVER)
	defer server.Close()

	for {
		client, err := server.Accept()
		if err != nil {
			log.Printf("FAILED TO ACCEPT CONNECTION FROM CLIENT: %v", err)
			continue
		}
		go initConn(client, conf.PSK)
	}
}

func initConf() *config.Server {
	path := flag.String("c", "./config.json", "CONFIG FILE PATH")
	flag.Parse()
	log.Printf("LOADING CONFIG FROM %v", *path)
	conf, err := config.LoadServerConf(*path)
	if err != nil {
		log.Fatalf("LOAD CONFIG ERROR: %v", err)
	}
	utils.HKEY1 = utils.SH256(conf.PSK[:10])
	utils.HKEY2 = utils.SH256(conf.PSK[24:])
	return conf
}

func initLsnr(addr string) net.Listener {
	defer log.Printf("LISTENER STARTED AT %v", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("LISTENER FAILED TO START: ", err)
	}
	return listener
}

func initConn(client net.Conn, PSK [32]byte) {
	eStream := utils.NewEncStream(client, &PSK)
	cStream := utils.NewSnappyStream(eStream)
	proxy.NewPServer(cStream).Forward()
}
