package main

import (
	"./compress"
	"./config"
	"./encrypt"
	"./proxy"
	"log"
	"net"
	"sync"
)

func main() {
	server := struct {
		conf *config.Server
		wg   sync.WaitGroup
	}{
		conf: config.LoadServerConf(),
		wg:   sync.WaitGroup{},
	}

	if len(server.conf.Tcpserver)*len(server.conf.Upstream) > 0 {
		tcp := initAddr("TCP SERVER", server.conf.Tcpserver)
		defer tcp.Close()
		server.wg.Add(1)
		go func() {
			for {
				defer server.wg.Done()
				src, err := tcp.Accept()
				if err != nil {
					log.Printf("FAILED TO ACCEPT TCP CONNECTION: %v", err)
					continue
				}
				dst, err := net.Dial("tcp", server.conf.Upstream)
				if err != nil {
					log.Printf("UPSTREAM SERVICE UNREACHABLE: %v", err)
					continue
				}
				go forward(src, dst, server.conf)
			}
		}()
	}

	if len(server.conf.Socksserver) > 0 {
		socks := initAddr("SOCKS SERVER", server.conf.Socksserver)
		defer socks.Close()
		server.wg.Add(1)
		go func() {
			defer server.wg.Done()
			for {
				client, err := socks.Accept()
				if err != nil {
					log.Printf("FAILED TO ACCEPT SOCKS CONNECTION: %v", err)
					continue
				}
				go socks5(client, server.conf)
			}
		}()
	}

	server.wg.Wait()
}

func initAddr(name, addr string) net.Listener {
	defer log.Printf("%s LISTENER STARTED ON %s", name, addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("LISTENER FAILED TO START: ", err)
	}
	return listener
}

func socks5(client net.Conn, conf *config.Server) {
	eStream := encrypt.NewEncStreamServer(client, conf.Getkeyring())
	switch conf.Compression {
	case "snappy":
		cStream := compress.NewSnappyStream(eStream)
		proxy.NewProxyServer(cStream).Connect()
	case "brotli":
		cStream := compress.NewBrotliStream(eStream)
		proxy.NewProxyServer(cStream).Connect()
	default:
		proxy.NewProxyServer(eStream).Connect()
	}
}

func forward(src, dst net.Conn, conf *config.Server) {
	eStream := encrypt.NewEncStreamServer(src, conf.Getkeyring())
	switch conf.Compression {
	case "snappy":
		cStream := compress.NewSnappyStreamServer(eStream)
		proxy.Pipe(cStream, dst)
	case "brotli":
		cStream := compress.NewBrotliStream(eStream)
		proxy.Pipe(cStream, dst)
	default:
		proxy.Pipe(eStream, dst)
	}
}
