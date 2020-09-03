package main

import (
	"bufio"
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
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
		fmt.Printf("Usage: ./%s sign|<request>\n", os.Args[0])
		return
	}

	if os.Args[1] == "sign" {
		req := interview()
		loadRSAKey(req)
		loadNonce(req)
		signRequest(req)
		writeNonce(req.nonce)
		fmt.Printf("x-sign: %s\n", req.signature)
		fmt.Printf("x-nonce: %s\n", req.nonce)

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
	f, err := os.Open("private.pem")
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
	n, err := os.Open("nonce")
	defer n.Close()
	req.nonce = "0"
	if err != nil {
		fmt.Println(err)
		return
	}

	nBytes, err := ioutil.ReadAll(n)
	if err != nil {
		panic(err)
	}
	nStr := strings.Trim(string(nBytes), "\n ")
	if nStr == "" {
		nStr = "0"
	}

	nonce, err := strconv.Atoi(nStr)
	if err != nil {
		panic(err)
	}

	req.nonce = strconv.Itoa(nonce + 1)
}

func writeNonce(nonce string) {
	n, err := os.OpenFile("nonce", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	defer n.Close()
	if err != nil {
		panic(err)
	}
	n.WriteString(nonce)
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
	k, err := os.Open("apikey")
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

func interview() *request {
	req := &request{}
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("url: ")
	text, _ := reader.ReadString('\n')
	req.uri = strings.Trim(text, "\n ")

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

	return req
}
