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
	conf := LoadConf()
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

func LoadConf() *config.Server {
	path := flag.String("c", "./config.json", "CONFIG FILE PATH")
	flag.Parse()
	log.Printf("LOADING CONFIG FROM %v", *path)
	conf, err := config.InitServer(*path)
	if err != nil {
		log.Fatalf("LOAD CONFIG ERROR: %v", err)
	}
	utils.HKEY1 = utils.SH256(conf.PSK[:10])
	utils.HKEY2 = utils.SH256(conf.PSK[20:])
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

func initConn(client net.Conn, conf *config.Server) {
	eStream := utils.NewEncStream(client, &conf.PSK)
	switch conf.COMPRESSION{
		case "none":
			proxy.NewPServer(eStream).Forward()
		case "S2":
			cStream := utils.NewS2Stream(eStream)
			proxy.NewPServer(cStream).Forward()
		case "snappy":
			cStream := utils.NewSnappyStream(eStream)
			proxy.NewPServer(cStream).Forward()
		default:
			cStream := utils.NewSnappyStream(eStream)
			proxy.NewPServer(cStream).Forward()	
	}
}
