package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	version = "1.0.0.0"
)

var (
	appkey      string
	accessToken string
	text        string
	format      string
	voice       string
	sampleRate  string
)

type ttsConf struct {
	localip   string `yaml: "LocalIP"`
	localport string `yaml: "LocalPort"`
}

func (conf *ttsConf) getTTSConf() *ttsConf {
	yamlFile, err := ioutil.ReadFile("Aliy_TTS.yaml")
	if err != nil {
		return nil
	}

	err = yaml.Unmarshal(yamlFile, conf)
	if err != nil {
		return nil
	}

	return conf
}

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "version" || strings.ToUpper(os.Args[1]) == "-V" || os.Args[1] == "-version" || os.Args[1] == "--version") {
		fmt.Printf("WIPCC_AliyunTTS_Go version: %s\n", version)
		return
	}
	fmt.Printf("WIPCC_AliyunTTS_Go version: %s\n", version)

	var conf ttsConf
	conf.getTTSConf()

	fmt.Printf("System running, Listening %s: %s\n", conf.localip, conf.localport)

	go initUDPServer()
	go doGetRequest()
}

func initUDPServer() {
	serverSock, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 8080,
	})
	if err != nil {
		fmt.Println("Listen failed!", err)
		return
	}
	defer serverSock.Close()

	for {
		data := make([]byte, 4096)
		read, remoteAddr, err := serverSock.ReadFromUDP(data)
		if err != nil {
			fmt.Println("read data failed!", err)
			continue
		}
		fmt.Println(read, remoteAddr)
		fmt.Printf("%s\n\n", data)

		senddata := []byte("hello client!")
		_, err = serverSock.WriteToUDP(senddata, remoteAddr)
		if err != nil {
			fmt.Println("send data failed!", err)
			return
		}
	}
}

func doGetRequest() {
	req := "https://nls-gateway.cn-shanghai.aliyuncs.com/stream/v1/tts"
	req += "?appkey=" + appkey
	req += "&token=" + accessToken
	req += "&text=" + text
	req += "&format=" + format
	req += "&voice=" + voice
	req += "&sampleRate=" + sampleRate

	fmt.Println(req)

	resp, err := http.Get(req)
	if err != nil {
		log.Print("Oops, err")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println(body)
}
