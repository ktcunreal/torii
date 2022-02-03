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
	client := struct {
		conf *config.Client
		wg   sync.WaitGroup
	}{
		conf: config.LoadClientConf(),
		wg:   sync.WaitGroup{},
	}

	if len(client.conf.Tcpserver)*len(client.conf.Tcpclient) > 0 {
		tcp := initAddr("TCP", client.conf.Tcpclient)
		defer tcp.Close()
		client.wg.Add(1)
		go func() {
			defer client.wg.Done()
			for {
				src, err := tcp.Accept()
				if err != nil {
					log.Printf("FAILED TO ACCEPT TCP CONNECTION: %v", err)
					continue
				}
				dst, err := net.Dial("tcp", client.conf.Tcpserver)
				if err != nil {
					log.Println("TCP SERVER UNREACHABLE")
					continue
				}
				go forward(src, dst, client.conf)
			}
		}()
	}

	if len(client.conf.Socksserver)*len(client.conf.Socksclient) > 0 {
		socks := initAddr("SOCKS CLIENT", client.conf.Socksclient)
		defer socks.Close()
		client.wg.Add(1)
		go func() {
			defer client.wg.Done()
			for {
				src, err := socks.Accept()
				if err != nil {
					log.Println("FAILED TO ACCEPT SOCKS CONNECTION: ", err)
					continue
				}
				dst, err := net.Dial("tcp", client.conf.Socksserver)
				if err != nil {
					log.Println("SOCKS SERVER UNREACHABLE: ", err)
					continue
				}
				go socks5(dst, src, client.conf)
			}
		}()
	}

	client.wg.Wait()
}

func initAddr(name, addr string) net.Listener {
	defer log.Printf("%s LISTENER STARTED AT %s", name, addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("LISTENER FAILED TO START: ", err)
	}
	return listener
}

func socks5(server, client net.Conn, conf *config.Client) {
	eStream := encrypt.NewEncStreamClient(server, conf.Getkeyring())
	switch conf.Compression {
	case "snappy":
		cStream := compress.NewSnappyStream(eStream)
		proxy.NewProxyClient(client).Connect(cStream)
	case "brotli":
		cStream := compress.NewBrotliStream(eStream)
		proxy.NewProxyClient(client).Connect(cStream)
	default:
		proxy.NewProxyClient(client).Connect(eStream)
	}
}

func forward(src, dst net.Conn, conf *config.Client) {
	eStream := encrypt.NewEncStreamClient(dst, conf.Getkeyring())
	switch conf.Compression {
	case "snappy":
		cStream := compress.NewSnappyStreamClient(eStream)
		proxy.Pipe(src, cStream)
	case "brotli":
		cStream := compress.NewBrotliStream(eStream)
		proxy.Pipe(src, cStream)
	default:
		proxy.Pipe(src, eStream)
	}
}
