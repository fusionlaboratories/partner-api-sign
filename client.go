package main

import (
	"bufio"
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var (
	flagURL        = flag.String("url", "", "URL")
	flagPEMFile    = flag.String("pemfile", "private.pem", "Private key pem file")
	flagAPIKeyFile = flag.String("keyfile", "apikey", "API key file")
	flagAPIKey     = flag.String("apikey", "", "API key")
	flagNonce      = flag.String("nonce", "", "Reset nonce")
	flagNonceFile  = flag.String("noncefile", "nonce", "Nonce file")
)

type request struct {
	method    string
	uri       string
	body      []byte
	apiKey    string
	nonce     string
	signature string
	rsaKey    *rsa.PrivateKey
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: ./%s sign|websocket|<request>\n", os.Args[0])
		flag.Usage()
		return
	}

	if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
		fmt.Printf("Usage: ./%s sign|websocket|<request>\n", os.Args[0])
		flag.Usage()
		return
	}

	switch os.Args[1] {
	case "sign":
		req := interview(true)
		loadRSAKey(req)
		loadNonce(req)
		signRequest(req)
		writeNonce(req.nonce)
		fmt.Printf("x-sign: %s\n", req.signature)
		fmt.Printf("x-nonce: %s\n", req.nonce)

		os.Exit(0)
	case "websocket":
		req := interview(false)
		loadRSAKey(req)
		loadNonce(req)
		loadAPIKey(req)
		signRequest(req)
		writeNonce(req.nonce)
		fmt.Printf("x-sign: %s\n", req.signature)
		fmt.Printf("x-nonce: %s\n", req.nonce)

		connectWebSocket(req)

		os.Exit(0)
	}

	req := loadRequest(os.Args[1])
	loadAPIKey(req)
	loadRSAKey(req)
	loadNonce(req)
	signRequest(req)
	sendRequest(req)
	writeNonce(req.nonce)
}

func loadRSAKey(req *request) {
	f, err := os.Open(*flagPEMFile)
	defer f.Close()

	if err != nil {
		panic(err)
	}

	pemData, err := ioutil.ReadAll(f)

	block, _ := pem.Decode(pemData)

	req.rsaKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}
}

func loadNonce(req *request) {
	var nStr = ""
	if *flagNonce == "" {
		nBytes, err := ioutil.ReadFile(*flagNonceFile)
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Println("Error loading nonce file", err)
				os.Exit(1)
			}

			fmt.Println("Nonce file not found")
			nStr = "0"
		} else {
			nStr = strings.TrimSpace(string(nBytes))
		}

	} else {
		nStr = *flagNonce
	}

	nonce, err := strconv.Atoi(nStr)
	if err != nil {
		fmt.Println("Error parse nonce", err)
		os.Exit(1)
	}

	req.nonce = strconv.Itoa(nonce + 1)
}

func writeNonce(nonce string) {
	if err := ioutil.WriteFile(*flagNonceFile, []byte(nonce), 0644); err != nil {
		fmt.Println("Error write nonce file", err)
		os.Exit(1)
	}
}

func loadRequest(req string) *request {
	uriFile, err := os.Open(fmt.Sprintf("requests/%s/uri", req))
	defer uriFile.Close()
	if err != nil {
		panic(err)
	}

	uriData, err := ioutil.ReadAll(uriFile)
	uriString := strings.Trim(string(uriData), "\n ")
	words := strings.Split(uriString, " ")
	if len(words) != 2 {
		panic(errors.New("invalid uri file"))
	}

	result := &request{
		method: words[0],
		uri:    words[1],
	}

	if result.method == http.MethodPost || result.method == http.MethodPut || result.method == http.MethodPatch {
		bodyFile, err := os.Open(fmt.Sprintf("requests/%s/body", req))
		if err != nil {
			panic(err)
		}

		body, err := ioutil.ReadAll(bodyFile)
		if err != nil {
			panic(err)
		}
		result.body = body
	}

	return result
}

func loadAPIKey(req *request) {
	if *flagAPIKey != "" {
		req.apiKey = *flagAPIKey
		return
	}

	k, err := os.Open(*flagAPIKeyFile)
	defer k.Close()
	if err != nil {
		panic(err)
	}

	key, err := ioutil.ReadAll(k)
	if err != nil {
		panic(key)
	}

	req.apiKey = base64.RawURLEncoding.EncodeToString(key)
}

func signRequest(req *request) {
	h := sha256.New()
	h.Write([]byte(req.nonce))
	h.Write([]byte(req.uri))
	h.Write(req.body)
	dgst := h.Sum(nil)
	fmt.Println(hex.EncodeToString(dgst))
	signature, err := rsa.SignPKCS1v15(nil, req.rsaKey, crypto.SHA256, dgst)
	if err != nil {
		panic(err)
	}

	req.signature = base64.RawURLEncoding.EncodeToString(signature)
}

func sendRequest(req *request) {
	client := http.Client{}
	httpReq, err := http.NewRequest(req.method, req.uri, bytes.NewReader(req.body))
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s %s\n", req.method, req.uri)
	fmt.Printf("x-api-key: %s\n", req.apiKey)
	fmt.Printf("x-sign: %s\n", req.signature)
	fmt.Printf("x-nonce: %s\n", req.nonce)
	fmt.Println("---")
	httpReq.Header.Add("x-api-key", req.apiKey)
	httpReq.Header.Add("x-sign", req.signature)
	httpReq.Header.Add("x-nonce", req.nonce)

	resp, err := client.Do(httpReq)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n%s\n", resp.Status, string(body))

}

func interview(requestBody bool) *request {
	req := &request{}
	reader := bufio.NewReader(os.Stdin)

	if *flagURL != "" {
		req.uri = *flagURL
		fmt.Println("url: ", req.uri)
	} else {
		fmt.Printf("url: ")
		text, _ := reader.ReadString('\n')
		req.uri = strings.Trim(text, "\n ")
	}

	if requestBody {
		fmt.Printf("body (hit return to end):\n")
		scanner := bufio.NewScanner(os.Stdin)
		buffer := ""
		for scanner.Scan() {
			if scanner.Text() == "" {
				break
			}
			buffer += scanner.Text() + "\n"
		}
		compacted := []byte{}
		b := bytes.NewBuffer(compacted)
		json.Compact(b, []byte(buffer))
		req.body = b.Bytes()
	}

	return req
}
