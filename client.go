package main

import (
	"bufio"
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	flagURL        = flag.String("url", "", "URL to be used")
	flagMethod     = flag.String("method", "", "HTTP method to be used")
	flagPEMFile    = flag.String("pem-file", "private.pem", "Private key pem file")
	flagAPIKeyFile = flag.String("key-file", "apikey", "API key file")
	flagAPIKey     = flag.String("api-key", "", "API key")
	flagTimestamp  = flag.String("timestamp", "", "Timestamp to be used")
)

type request struct {
	method    string
	uri       string
	body      []byte
	apiKey    string
	timestamp string
	signature string
	rsaKey    *rsa.PrivateKey
}

func checkErr(err error) {
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func loadRSAKey(req *request) error {
	f, err := os.Open(*flagPEMFile)
	defer f.Close()

	if err != nil {
		return errors.Wrap(err, "load RSA key")
	}

	pemData, err := ioutil.ReadAll(f)

	block, _ := pem.Decode(pemData)

	req.rsaKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return errors.Wrap(err, "parse RSA key")
	}
	return nil
}

func genTimestamp(req *request) {
	if *flagTimestamp != "" {
		req.timestamp = strings.TrimSpace(*flagTimestamp)
	} else {
		req.timestamp = fmt.Sprintf("%v", time.Now().Unix())
	}
}

func loadRequest(req *request, requestName string) error {
	if *flagURL != "" {
		if *flagMethod == "" {
			return errors.New("URL option provided without method")
		}
		req.uri = *flagURL
		req.method = *flagMethod
	} else {
		uriFile, err := os.Open(fmt.Sprintf("requests/%s/uri", requestName))
		defer uriFile.Close()
		if err != nil {
			return errors.Wrap(err, "load request URI file")
		}

		uriData, err := ioutil.ReadAll(uriFile)
		uriString := strings.TrimSpace(string(uriData))
		words := strings.Split(uriString, " ")
		if len(words) != 2 {
			return errors.New("invalid URI file")
		}

		req.method = words[0]
		req.uri = words[1]
	}

	if req.method == http.MethodPost || req.method == http.MethodPut || req.method == http.MethodPatch {
		bodyFile, err := os.Open(fmt.Sprintf("requests/%s/body", requestName))
		if err != nil {
			return errors.Wrap(err, "open body file")
		}

		body, err := ioutil.ReadAll(bodyFile)
		if err != nil {
			return errors.Wrap(err, "read body file")
		}
		req.body = body
	}

	return nil
}

func loadAPIKey(req *request) error {
	if *flagAPIKey != "" {
		req.apiKey = *flagAPIKey
		return nil
	}

	k, err := os.Open(*flagAPIKeyFile)
	defer k.Close()
	if err != nil {
		return errors.Wrap(err, "open api key file")

	}

	key, err := ioutil.ReadAll(k)
	if err != nil {
		return errors.Wrap(err, "read api key file")
	}

	req.apiKey = strings.TrimSpace(string(key))
	return nil
}

func signRequest(req *request) error {
	h := sha256.New()
	h.Write([]byte(req.timestamp))
	h.Write([]byte(req.uri))
	h.Write(req.body)
	dgst := h.Sum(nil)
	signature, err := rsa.SignPKCS1v15(nil, req.rsaKey, crypto.SHA256, dgst)
	if err != nil {
		return errors.Wrap(err, "sign request")
	}

	req.signature = base64.RawURLEncoding.EncodeToString(signature)
	return nil
}

func sendRequest(req *request) error {
	client := http.Client{}
	httpReq, err := http.NewRequest(req.method, req.uri, bytes.NewReader(req.body))
	if err != nil {
		return errors.Wrap(err, "create http request")
	}

	fmt.Printf("%s %s\n", req.method, req.uri)
	fmt.Printf("x-api-key: %s\n", req.apiKey)
	fmt.Printf("x-sign: %s\n", req.signature)
	fmt.Printf("x-timestamp: %s\n", req.timestamp)
	fmt.Println("---")
	httpReq.Header.Add("x-api-key", req.apiKey)
	httpReq.Header.Add("x-sign", req.signature)
	httpReq.Header.Add("x-timestamp", req.timestamp)

	resp, err := client.Do(httpReq)
	if err != nil {
		return errors.Wrap(err, "execute http request")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "read response body")
	}
	fmt.Printf("%s\n%s\n", resp.Status, string(body))
	return nil
}

func interview(req *request, requestBody bool) error {
	reader := bufio.NewReader(os.Stdin)

	if *flagURL != "" {
		req.uri = *flagURL
		fmt.Println("url: ", req.uri)
	} else {
		fmt.Printf("url: ")
		text, _ := reader.ReadString('\n')
		req.uri = strings.TrimSpace(text)
	}

	terminatorMessage := "hit Ctrl+D (twice if body has no trailing new line) to end"
	if runtime.GOOS == "windows" {
		terminatorMessage = "hit Ctrl+Z followed by <enter> to end"
	}
	if requestBody {
		fmt.Printf("body (%s):\n", terminatorMessage)
		reader := bufio.NewReader(os.Stdin)
		readBuffer := make([]byte, 10024)
		var bodyBuffer []byte
		for {
			n, err := reader.Read(readBuffer)
			if err != nil {
				if err == io.EOF {
					break
				}
			}
			if n == 0 {
				continue
			}
			if n > 1 {
				if readBuffer[n-2] == '\r' && readBuffer[n-1] == '\n' {
					readBuffer[n-2] = '\n'
					n--
				}
			}
			bodyBuffer = append(bodyBuffer, readBuffer[:n]...)
		}

		if len(bodyBuffer) != 0 {
			req.body = bodyBuffer
		}
	}

	return nil
}

func printHelpAndExit() {
	fmt.Printf("Usage: %s sign|websocket|liquidityhub|<request> [options]\n\nOptions:\n", filepath.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		printHelpAndExit()
	}

	if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
		printHelpAndExit()
	}

	req := &request{}

	switch os.Args[1] {
	case "sign":
		checkErr(interview(req, true))
		checkErr(loadRSAKey(req))
		genTimestamp(req)
		checkErr(signRequest(req))
		fmt.Printf("x-sign: %s\n", req.signature)
		fmt.Printf("x-timestamp: %s\n", req.timestamp)

	case "websocket", "coreclient", "liquidityhub", "walletupdates":
		checkErr(interview(req, false))
		checkErr(loadRSAKey(req))
		genTimestamp(req)
		checkErr(loadAPIKey(req))
		checkErr(signRequest(req))
		fmt.Printf("x-sign: %s\n", req.signature)
		fmt.Printf("x-timestamp: %s\n", req.timestamp)

		var wsType int
		switch os.Args[1] {
		case "coreclient":
			wsType = wsCoreClient
		case "liquidityhub":
			wsType = wsLiquidityHub
		case "walletupdates":
			wsType = wsWalletUpdates
		default:
			wsType = wsRaw
		}

		connectWebSocket(req, wsType)
	default:
		checkErr(loadRequest(req, os.Args[1]))
		checkErr(loadAPIKey(req))
		checkErr(loadRSAKey(req))
		genTimestamp(req)
		checkErr(signRequest(req))
		checkErr(sendRequest(req))
	}
}
