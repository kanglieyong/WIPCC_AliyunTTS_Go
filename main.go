package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

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
	LocalIP   string `yaml: "LocalIP"`
	LocalPort int    `yaml: "LocalPort"`
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

	fmt.Printf("System running, Listening %s: %d\n", conf.LocalIP, conf.LocalPort)

	go initUDPServer(&conf)
	go doGetRequest()
}

func initUDPServer(conf *ttsConf) {
	addr := net.UDPAddr{
		IP:   net.ParseIP(conf.LocalIP),
		Port: conf.LocalPort,
	}
	conn, err := net.ListenUDP("udp4", &addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	var buf [1024]byte
	for {
		read, remoteAddr, err := conn.ReadFromUDP(buf[:])
		if err != nil {
			log.Print(err)
			continue
		}
		fmt.Println(read, remoteAddr)
		fmt.Printf("%s\n\n", buf[:])

		senddata := []byte("hello client!")
		_, err = conn.WriteToUDP(senddata, remoteAddr)
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

	dst, err := os.Create(time.Now().Format("2006-01-02_15_04_05"))
	if err != nil {
		log.Fatal(err)
	}
	defer dst.Close()

	wlen, err := io.Copy(dst, resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("write len=%d\n", wlen)
}
