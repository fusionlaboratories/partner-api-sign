package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsRaw = iota
	wsCoreClient
	wsLiquidityHub
	wsWalletUpdates
)

type Parser interface {
	Parse() string
}

type ActionInfo struct {
	ID         string `json:"id"`
	ClientID   string `json:"clientID"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Timestamp  int64  `json:"timestamp"`
	ExpireTime int64  `json:"expireTime"`
}

func (a *ActionInfo) Parse() string {
	return fmt.Sprintf("NEW ACTION: %v\nType: %v\tStatus: %v\nCreated: %v\tExpires: %v\n\n", a.ID, a.Type, a.Status, time.Unix(a.Timestamp, 0).Format(time.RFC822), time.Unix(a.ExpireTime, 0).Format(time.RFC822))
}

type LiquidityHubInfo struct {
	TxID   string `json:"txID"`
	Status string `json:"status"`
}

func (lh *LiquidityHubInfo) Parse() string {
	return fmt.Sprintf("UPDATE IN LIQUIDITY HUB. TxID: %v, Status: %v", lh.TxID, lh.Status)
}

type RawInfo map[string]interface{}

func (ri *RawInfo) Parse() string {
	out, _ := json.MarshalIndent(ri, "", "  ")
	return string(out)
}

func connectWebSocket(req *request, wsType int) {
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
			var v Parser
			switch wsType {
			case wsCoreClient:
				v = &ActionInfo{}
			case wsLiquidityHub:
				v = &LiquidityHubInfo{}
			default:
				v = &RawInfo{}
			}
			if err := c.ReadJSON(v); err != nil {
				fmt.Println("read from websocket:", err)
				return
			}
			fmt.Println(v.Parse())
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
