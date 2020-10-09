package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

type ActionInfo struct {
	ID         string `json:"id"`
	ClientID   string `json:"clientID"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Timestamp  int64  `json:"timestamp"`
	ExpireTime int64  `json:"expireTime"`
}

func connectWebSocket(req *request) {
	url := req.uri

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	fmt.Printf("connecting to %s\n", url)

	headers := http.Header{}
	headers.Add("x-api-key", req.apiKey)
	headers.Add("x-sign", req.signature)
	headers.Add("x-nonce", req.nonce)

	c, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		fmt.Println("cannot connect to websocket:", err)
		return
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			v := &ActionInfo{}
			if err := c.ReadJSON(v); err != nil {
				fmt.Println("read from websocket:", err)
				return
			}

			fmt.Printf("recv: %v\n", v)
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			fmt.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
