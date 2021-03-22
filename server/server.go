package main

import (
	"../config"
	"../utils"
	"flag"
	"log"
	"net"
)

func main() {
	conf := initConf()
	utils.M.SetKey(conf.PSK[:4])

	server := initLsnr(conf.SERVER)
	defer server.Close()

	for {
		client, err := server.Accept()
		if err != nil {
			log.Printf("FAILED TO ACCEPT CONNECTION FROM CLIENT: %v", err)
			continue
		}
		initConn(client, conf.PSK)
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

func initConf() *config.Server {
	path := flag.String("c", "./config.json", "CONFIGURATION FILE PATH")
	flag.Parse()
	log.Printf("LOADING CONFIGURATION FROM %v", *path)
	conf, err := config.LoadSC(*path)
	if err != nil {
		log.Fatalf("LOAD CONFIGURATION ERROR: %v", err)
	}
	return conf
}

func initConn(client net.Conn, PSK [32]byte) {
	eConn := utils.NewEncryptedStream(client, &PSK)
	cConn := utils.NewCompStream(eConn)
	go utils.NewSocks5Server(cConn).Proxy()
}
