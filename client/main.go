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
	listener := initLsnr(conf.CLIENT)
	defer listener.Close()

	for {
		client, err := listener.Accept()
		if err != nil {
			log.Println("FAILED TO ACCEPT CONNECTION FROM USER: ", err)
			continue
		}
		server, err := net.Dial("tcp", conf.SERVER)
		if err != nil {
			log.Println("COULD NOT CONNECT TO SERVER: ", err)
			continue
		}
		go initConn(server, client, &conf.PSK)
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

func initConf() *config.Client {
	path := flag.String("c", "./config.json", "CONFIGURATION FILE PATH")
	flag.Parse()
	log.Printf("LOADING CONFIGURATION FROM %v", *path)
	conf, err := config.LoadClientConf(*path)
	if err != nil {
		log.Fatalf("LOAD CONFIGURATION ERROR: %v", err)
	}
	utils.HKEY1 = utils.SH256(conf.PSK[:10])
	utils.HKEY2 = utils.SH256(conf.PSK[24:])
	return conf
}

func initConn(server, client net.Conn, PSK *[32]byte) {
	eStream := utils.NewEncStream(server, PSK)
	cStream := utils.NewSnappyStream(eStream)
	proxy.NewPClient(client).Forward(cStream)
}
