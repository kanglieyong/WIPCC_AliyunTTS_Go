package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	VERSION = "1.0.0.0"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "version" || strings.ToUpper(os.Args[1]) == "-V" || os.Args[1] == "-version" || os.Args[1] == "--version") {
		fmt.Printf("WIPCC_AliyunTTS_Go version: %s\n", VERSION)
		return
	}

	server_sock, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 8080,
	})
	if err != nil {
		fmt.Println("Listen failed!", err)
		return
	}
	defer server_sock.Close()

	for {
		data := make([]byte, 4096)
		read, remoteAddr, err := server_sock.ReadFromUDP(data)
		if err != nil {
			fmt.Println("read data failed!", err)
			continue
		}
		fmt.Println(read, remoteAddr)
		fmt.Printf("%s\n\n", data)

		senddata := []byte("hello client!")
		_, err = server_sock.WriteToUDP(senddata, remoteAddr)
		if err != nil {
			fmt.Println("send data failed!", err)
			return
		}
	}
}
