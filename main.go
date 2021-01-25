package main

import (
	"bufio"
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

	//return
	ttsRequest := make(chan string)
	go getTTSPost(ttsRequest)
	//go getTTSResult(ttsRequest)

	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		msg := input.Text()
		if msg == "quit" {
			break
		}
		ttsRequest <- msg
	}

	time.Sleep(5)
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
		reqBuilder.WriteString(`&sampleRate=`)
		reqBuilder.WriteString(strconv.Itoa(ttsConf.SampleRate))

		dst, err := os.Create(time.Now().Format(`2006-01-02_15_04_05`) + `.` + ttsConf.Format)
		if err != nil {
			fmt.Println("error: os.Create")
			log.Fatal(err)
		}
		defer dst.Close()

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

		wlen, err := io.Copy(dst, resp.Body)
		if err != nil {
			fmt.Println("error: io.Copy")
			log.Fatal(err)
		}
		fmt.Printf("write len=%d\n", wlen)

		return
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("error: ioutil.ReadAll")
			log.Fatal(err)
		}
		dataLen := len(contents)
		fmt.Printf("write len=%d\n", dataLen)

		//postData := make([]byte, dataLen/2, dataLen/2)

		convert16to8(contents, dataLen)


	}
}

func getTTSPost(ttsRequest chan string) {
	host := `nls-gateway.cn-shanghai.aliyuncs.com`
    url := `https://` + host + `/stream/v1/tts`
	for {
		textUrl := <-ttsRequest
		fmt.Println(textUrl)

		reqBuilder := strings.Builder{}
		reqBuilder.WriteString(`https://`)
		reqBuilder.WriteString(host)
		reqBuilder.WriteString(`/stream/v1/tts`)
        postHeader := reqBuilder.String()

        client := http.Client{}



        req, err := http.NewRequest("POST", postHeader, bytes.NewReader(contents))
        req.Header.Add(`X-NLS-Token`, ttsConf.AccessToken)
        req.Header.Add(`Content-type`, `application/octet-stream`)
        req.Header.Add(`Content-Length`, strconv.Itoa(contentLen))
        req.Header.Add(`Host`, `nls-gateway.cn-shanghai.aliyuncs.com`)


		escapeUrl := url.QueryEscape(textUrl)






	}
}

func convert16to8(preData []byte, dataLen int) {
	postData := make([]byte, dataLen/2, dataLen/2)
	var counter int

	for pos := 0; pos < dataLen; pos += 2 {
		counter++
		data := make([]byte, 2, 2)
		data = preData[pos : pos+2]

		frame := int16(data[0])
		frame = (frame << 8)
		frame += int16(data[1])

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
		postData = append(postData, b)
	}

	err := ioutil.WriteFile("test.pcm", postData, 0666)
	if err != nil {
		fmt.Println("error: ioutil.WriteFile")
		log.Fatal(err)
	}
}
