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
		err = ioutil.WriteFile(fileName + `.pcm`, contents, 0664)
		if err != nil {
			fmt.Println("error: ioutil.WriteFile")
			log.Fatal(err)
		}

		dataLen := preDataLen / 2
		postData := make([]byte, dataLen, dataLen)
		convert16to8(contents, postData, dataLen)

		err = ioutil.WriteFile(fileName + `_post.pcm`, postData, 0664)
		if err != nil {
			fmt.Println("error: ioutil.WriteFile")
			log.Fatal(err)
		}
	}
}

func convert16to8(preData, postData []byte, dataLen int) {
	var counter int

	for pos := 0; pos < dataLen; pos += 1 {
		counter++
		data := make([]byte, 2, 2)
		data = preData[2*pos : 2*pos+2]

		frame := int16(data[1])
		frame = (frame << 8)
		frame += int16(data[0])

		var a uint16 // A-law value we are forming
		var b byte

		// -ve value
		// Note, ones compliment is used here as this keeps encoding symetrical
		// and equal spaced around zero cross-over, (it also matches the standard).
		if frame < 0 {
			frame = ^frame
			a = 0x00 // sign = 0
		} else {
			// +ve value
			a = 0x80 // sign = 1
		}

		// Calculate segment and interval numbers
		frame = (frame >> 4)
		if frame > 0x20 {
			if frame >= 0x100 {
				frame = (frame >> 4)
				a += 0x40
			}

			if frame >= 0x40 {
				frame = (frame >> 2)
				a += 0x20
			}

			if frame >= 0x20 {
				frame = (frame >> 1)
				a += 0x10
			}
		}
		// a&0x70 now holds segment value and 'p' the interval number

		a += uint16(frame) // a now equal to encoded A-law value
		a = a ^ 0x55
		b = byte(a)
		if counter % 1000 == 0 {
			fmt.Println(b)
		}
		//postData = append(postData, b)
		postData[pos] = b
	}
}
