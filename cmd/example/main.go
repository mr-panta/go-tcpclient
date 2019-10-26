package main

import (
	"log"
	"net"
	"time"

	"github.com/mr-panta/go-tcpclient"
)

var data []byte

func main() {
	log.SetFlags(log.Lshortfile)
	genData(10000000)
	go runHost()
	time.Sleep(time.Second)
	client, err := tcpclient.NewClient("127.0.0.1:3000", 1, 1, 100*time.Millisecond, 10*time.Millisecond, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		output, err := client.Send(data)
		if err != nil {
			log.Fatal(err)
		}
		if compare(data, output) {
			log.Println("RECEIVE: PASS")
		} else {
			log.Println("RECEIVE: NOT PASS")
		}
	}
	client.Close()
}

func runHost() {
	listener, err := net.Listen("tcp", "127.0.0.1:3000")
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
		}
		go func() {
			for {
				err = tcpclient.Reader(conn, process)
				if err != nil {
					log.Println(err)
					break
				}
			}
		}()
	}
}

func process(input []byte) (output []byte, err error) {
	if compare(data, input) {
		log.Println("SEND: PASS")
	} else {
		log.Println("SEND: NOT PASS")
	}
	return input, nil
}

func compare(a, b []byte) bool {
	if len(a) != len(b) {
		log.Printf("a:%d b:%d", len(a), len(b))
		return false
	}
	cnt := 0
	for i := range a {
		if a[i] != b[i] {
			log.Printf("i:%d a:%d b:%d", i, a[i], b[i])
			cnt++
			if cnt > 0 {
				break
			}
		}
	}
	return cnt == 0
}

func genData(n int) {
	data = make([]byte, n)
	for i := 0; i < n; i++ {
		data[i] = byte(i % n)
	}
}
