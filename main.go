package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	version       = `1.0.0.0`
	maxBufferSize = 1024
)

type TTSConf struct {
	AppKey      string `yaml:"app_key"`
	AccessToken string `yaml:"access_token"`
	Format      string `yaml:"format"`
	Voice       string `yaml:"voice"`
	SampleRate  int    `yaml:"sample_rate"`
}

var (
	ttsConf TTSConf
)

func main() {
	fmt.Printf("WIPCC_AliyunTTS_Go version: %s\n", version)

	yamlFile, err := ioutil.ReadFile("conf/TTSConfig.yaml")
	if err != nil {
		fmt.Println(err)
		return
	}

	err = yaml.Unmarshal(yamlFile, &ttsConf)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%d\n", ttsConf.SampleRate)
	fmt.Println(ttsConf.AccessToken)

	return
	ttsRequest := make(chan string)
	go getTTSResult(ttsRequest)
}

func server(ctx context.Context, address string) (err error) {
	pc, err := net.ListenPacket("udp", address)
	if err != nil {
		return
	}
	defer pc.Close()

	doneChan := make(chan error, 1)
	buffer := make([]byte, maxBufferSize)

	// Given that waiting for packets to arrive is blocking by nature and we want
	// to be able of canceling such action if desired, we do that in a separate
	// go routine.
	go func() {
		for {
			n, addr, err := pc.ReadFrom(buffer)
			if err != nil {
				doneChan <- err
				return
			}

			fmt.Printf("packet-received: bytes=%d from=%s\n", n, addr.String())
			//reqStr := string(buffer[:n])
			//ttsRequest <- reqStr

			// Setting a deadline for the `write` operation allows us to not block
			// for longer than a specific timeout.
			//
			// In the case of a write operation, that'd mean waiting for the send
			// queue to be freed enough so that we are able to proceed.
			// deadline := time.Now().Add(*timeout)
			// err = pc.SetWriteDeadline(deadline)
			// if err != nil {
			// 	doneChan <- err
			// 	return
			// }

			n, err = pc.WriteTo(buffer[:n], addr)
			if err != nil {
				doneChan <- err
				return
			}

			fmt.Printf("packet-written: bytes=%d to=%s\n", n, addr.String())
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Println("cancelled")
		err = ctx.Err()
	case err = <-doneChan:
	}

	return
}

func getTTSResult(ttsRequest chan string) {
	textUrl := <-ttsRequest
	escapeUrl := url.QueryEscape(textUrl)

	reqBuilder := strings.Builder{}
	reqBuilder.WriteString(`https://nls-gateway.cn-shanghai.aliyuncs.com/stream/v1/tts`)
	reqBuilder.WriteString(`?appkey=`)
	reqBuilder.WriteString(ttsConf.AppKey)
	reqBuilder.WriteString(`&token=`)
	reqBuilder.WriteString(ttsConf.AccessToken)
	reqBuilder.WriteString(`&text=`)
	reqBuilder.WriteString(escapeUrl)
	reqBuilder.WriteString(`&format=`)
	reqBuilder.WriteString(ttsConf.Format)
	reqBuilder.WriteString(`&voice=`)
	reqBuilder.WriteString(ttsConf.Voice)
	reqBuilder.WriteString(`&sampleRate=`)
	reqBuilder.WriteString(strconv.Itoa(ttsConf.SampleRate))

	dst, err := os.Create(time.Now().Format(`2006-01-02_15_04_05`) + `.wav`)
	if err != nil {
		log.Fatal(err)
	}
	defer dst.Close()

	resp, err := http.Get(reqBuilder.String())
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}

	wlen, err := io.Copy(dst, resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("write len=%ld\n", wlen)
}
