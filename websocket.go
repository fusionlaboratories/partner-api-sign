package main

import (
	"fmt"
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
	headers.Add("x-timestamp", req.timestamp)

	c, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		fmt.Println("cannot connect to websocket:", err)
		return
	}
	defer c.Close()

	done := make(chan struct{})

	fmt.Printf("connected \n\n")
	go func() {
		defer close(done)
		for {
			v := &ActionInfo{}
			if err := c.ReadJSON(v); err != nil {
				fmt.Println("read from websocket:", err)
				return
			}

			fmt.Printf("NEW ACTION: %v\nType: %v\tStatus: %v\nCreated: %v\tExpires: %v\n\n", v.ID, v.Type, v.Status, time.Unix(v.Timestamp, 0).Format(time.RFC822), time.Unix(v.ExpireTime, 0).Format(time.RFC822))
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			fmt.Println("interrupt")

			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				fmt.Println("write close:", err)
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
