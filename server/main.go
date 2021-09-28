package main

import (
	"../config"
	"../proxy"
	"../utils"
	"log"
	"net"
	"sync"
)

func main() {
	conf := config.LoadServerConf()
	key := utils.NewKey(conf.RAW)
	wg := sync.WaitGroup{}

	if len(conf.TCPSERVER)*len(conf.UPSTREAM) > 0 {
		tcp := initAddr("TCP SERVER", conf.TCPSERVER)
		defer tcp.Close()
		wg.Add(1)
		go func() {
			for {
				defer wg.Done()
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

	if len(conf.SOCKSSERVER) > 0 {
		socks := initAddr("SOCKS SERVER", conf.SOCKSSERVER)
		defer socks.Close()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				client, err := socks.Accept()
				if err != nil {
					log.Printf("SOCKS SERVER FAILED TO ACCEPT CONNECTION: %v", err)
					continue
				}
				go socks5(client, conf, key)
			}
		}()
	}

	wg.Wait()
}

func initAddr(name, addr string) net.Listener {
	defer log.Printf("%s LISTENER STARTED ON %s", name, addr)
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
