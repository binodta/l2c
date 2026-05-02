package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	serverAddr string
	tunnelID   string
	localAddr  string
	token      string
	conn       *websocket.Conn
	mu         sync.Mutex
	stopChan   chan struct{}
}

type ProxyRequest struct {
	Type    string            `json:"type"`
	ID      string            `json:"id"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    *string           `json:"body"`
}

type ProxyResponse struct {
	Type    string            `json:"type"`
	ID      string            `json:"id"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    *string           `json:"body"`
}

func NewClient(serverAddr, tunnelID, localAddr, token string) *Client {
	return &Client{
		serverAddr: serverAddr,
		tunnelID:   tunnelID,
		localAddr:  localAddr,
		token:      token,
		stopChan:   make(chan struct{}),
	}
}

func (c *Client) Start() error {
	// Check if local server is running
	if err := c.checkLocalServer(); err != nil {
		return err
	}

	u := url.URL{Scheme: "wss", Host: c.serverAddr, Path: fmt.Sprintf("/connect/%s", c.tunnelID)}
	log.Printf("Connecting to %s", u.String())

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	header := http.Header{}
	header.Set("User-Agent", "l2c-proxy-client/1.0")
	if c.token != "" {
		header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	conn, resp, err := dialer.Dial(u.String(), header)
	if err != nil {
		if resp != nil {
			log.Printf("Dial failed with status: %d", resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			log.Printf("Response body: %s", string(body))
		}
		return fmt.Errorf("dial: %v", err)
	}
	c.conn = conn
	log.Printf("Connected successfully!")

	go c.listen()
	return nil
}

func (c *Client) listen() {
	defer c.conn.Close()

	for {
		select {
		case <-c.stopChan:
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Printf("read error: %v", err)
				return
			}

			var req ProxyRequest
			if err := json.Unmarshal(message, &req); err != nil {
				log.Printf("json unmarshal error: %v", err)
				continue
			}

			if req.Type == "req" {
				go c.handleRequest(req)
			}
		}
	}
}

func (c *Client) handleRequest(req ProxyRequest) {
	var body io.Reader
	if req.Body != nil {
		data, err := base64.StdEncoding.DecodeString(*req.Body)
		if err != nil {
			log.Printf("base64 decode error: %v", err)
			return
		}
		body = bytes.NewReader(data)
	}

	localURL := c.localAddr + req.URL
	httpReq, err := http.NewRequest(req.Method, localURL, body)
	if err != nil {
		log.Printf("create request error: %v", err)
		return
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		log.Printf("do request error: %v", err)
		c.sendResponse(ProxyResponse{
			Type:   "res",
			ID:     req.ID,
			Status: 502,
			Body:   nil,
		})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	respBodyEncoded := base64.StdEncoding.EncodeToString(respBody)

	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = v[0]
	}

	c.sendResponse(ProxyResponse{
		Type:    "res",
		ID:      req.ID,
		Status:  resp.StatusCode,
		Headers: headers,
		Body:    &respBodyEncoded,
	})
}

func (c *Client) checkLocalServer() error {
	u, err := url.Parse(c.localAddr)
	if err != nil {
		return fmt.Errorf("invalid local address: %v", err)
	}

	host := u.Host
	if host == "" {
		return fmt.Errorf("invalid local address: host is empty")
	}

	// Try to connect to the host
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return fmt.Errorf("local server at %s is not reachable. Make sure your local app is running on this port", c.localAddr)
	}
	conn.Close()
	return nil
}

func (c *Client) sendResponse(res ProxyResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, _ := json.Marshal(res)
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("write message error: %v", err)
	}
}

func (c *Client) Stop() {
	close(c.stopChan)
	if c.conn != nil {
		c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
	}
}
