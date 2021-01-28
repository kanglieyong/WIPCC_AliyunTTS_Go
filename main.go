package main

import (
	"bufio"
	"context"
	"fmt"
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
	A_LAW         = 1
	S16_LE        = 2
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
	fmt.Printf("AliyunTTS version: %s\n", version)

	if len(os.Args) > 1 {
		targetType, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Println("error: strconv.Atoi")
		}
		readContent(os.Args[1], os.Args[2], targetType)
		return
	}
	yamlFile, err := ioutil.ReadFile("config/TTSConfig.yaml")
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

	ttsRequest := make(chan string)

	go getTTSResult(ttsRequest)

	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		msg := input.Text()
		if msg == "quit" {
			break
		}
		ttsRequest <- msg
	}

	fmt.Println("quit main goroutine!!!")
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
	for {
		textUrl := <-ttsRequest
		fmt.Println(textUrl)
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
		reqBuilder.WriteString(`&sample_rate=`)
		reqBuilder.WriteString(strconv.Itoa(ttsConf.SampleRate))

		resp, err := http.Get(reqBuilder.String())
		if err != nil {
			fmt.Println("error: http.Get")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Println("error: resp.StatusCode")
			return
		}

		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("error: ioutil.ReadAll")
			log.Fatal(err)
		}
		preDataLen := len(contents)
		fmt.Printf("write len=%d\n", preDataLen)

		fileName := time.Now().Format(`2006-01-02_15_04_05`)
		err = ioutil.WriteFile(fileName+`.pcm`, contents, 0664)
		if err != nil {
			fmt.Println("error: ioutil.WriteFile")
			log.Fatal(err)
		}

		dataLen := preDataLen / 2
		postData := make([]byte, dataLen, dataLen)
		convert16to8(contents, postData, dataLen)

		err = ioutil.WriteFile(fileName+`_post.pcm`, postData, 0664)
		if err != nil {
			fmt.Println("error: ioutil.WriteFile")
			log.Fatal(err)
		}
	}
}

func readContent(pcmName, postName string, targetType int) {
	pcmContents, err := ioutil.ReadFile(pcmName)
	if err != nil {
		fmt.Println("error: ioutil.ReadFile")
		log.Fatal(err)
	}

	dataLen := len(pcmContents)
	var postDataLen int

	if targetType == A_LAW {
		postDataLen = dataLen / 2
		postContents := make([]byte, postDataLen, postDataLen)
		convert16to8(pcmContents, postContents, postDataLen)

		err = ioutil.WriteFile(postName+`_8k8bit_alaw.pcm`, postContents, 0664)
		if err != nil {
			fmt.Println("error: ioutil.WriteFile")
			log.Fatal(err)
		}
	} else if targetType == S16_LE {
		postDataLen = 2 * dataLen
		postContents := make([]byte, postDataLen, postDataLen)
		convert8to16(pcmContents, postContents, dataLen)

		err = ioutil.WriteFile(postName+`_8k16bit.pcm`, postContents, 0664)
		if err != nil {
			fmt.Println("error: ioutil.WriteFile")
			log.Fatal(err)
		}
	}
}
