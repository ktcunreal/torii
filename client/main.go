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
		go initConn(server, client, conf)
	}
}

func LoadConf() *config.Client {
	path := flag.String("c", "./config.json", "CONFIG FILE PATH")
	flag.Parse()
	log.Printf("LOADING CONFIG FROM %v", *path)
	conf, err := config.InitClient(*path)
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

func initConn(server, client net.Conn, conf *config.Client) {
	eStream := utils.NewEncStream(server, &conf.PSK)
	switch conf.COMPRESSION {
		case "none":
			proxy.NewPClient(client).Forward(eStream)
		case "LZ4": 
			cStream := utils.NewLZ4Stream(eStream)
			proxy.NewPClient(client).Forward(cStream)
		case "snappy":
			cStream := utils.NewSnappyStream(eStream)
			proxy.NewPClient(client).Forward(cStream)
		default:
			cStream := utils.NewSnappyStream(eStream)
			proxy.NewPClient(client).Forward(cStream)
	}
}
