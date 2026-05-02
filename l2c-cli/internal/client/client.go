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

const (
	maxBackoff     = 30 * time.Second
	initialBackoff = 1 * time.Second
)

type Client struct {
	serverAddr string
	tunnelID   string
	localAddr  string
	token      string
	conn       *websocket.Conn
	mu         sync.Mutex
	stopChan   chan struct{}
	stopped    bool
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

// Start connects and automatically reconnects on failure until Stop() is called.
func (c *Client) Start() error {
	backoff := initialBackoff
	attempt := 0

	for {
		// Check if we've been asked to stop
		select {
		case <-c.stopChan:
			return nil
		default:
		}

		if attempt > 0 {
			log.Printf("⟳  [%s] Reconnecting in %s... (attempt %d)", c.tunnelID, backoff, attempt)
			select {
			case <-c.stopChan:
				return nil
			case <-time.After(backoff):
			}
			// Exponential backoff, capped at maxBackoff
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}

		err := c.connect()
		if err != nil {
			// Fatal errors: don't retry
			if isFatal(err) {
				log.Printf("✗  [%s] Fatal error: %v", c.tunnelID, err)
				return err
			}
			log.Printf("✗  [%s] Connection error: %v", c.tunnelID, err)
			attempt++
			continue
		}

		// Connected — reset backoff
		backoff = initialBackoff
		attempt = 0

		// Block until connection drops or stop is called
		c.listen()

		// Check if stopped intentionally
		select {
		case <-c.stopChan:
			return nil
		default:
			log.Printf("⚠  [%s] Connection lost. Will reconnect...", c.tunnelID)
			attempt++
		}
	}
}

func (c *Client) connect() error {
	u := url.URL{Scheme: "wss", Host: c.serverAddr, Path: fmt.Sprintf("/connect/%s", c.tunnelID)}

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}

	header := http.Header{}
	header.Set("User-Agent", "l2c-proxy-client/1.0")
	if c.token != "" {
		header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	conn, resp, err := dialer.Dial(u.String(), header)
	if err != nil {
		if resp != nil {
			switch resp.StatusCode {
			case http.StatusUnauthorized, http.StatusForbidden:
				return &fatalError{fmt.Sprintf("Authentication failed (HTTP %d). Check your token in ~/.l2c/config.json", resp.StatusCode)}
			case http.StatusNotFound:
				return &fatalError{fmt.Sprintf("Tunnel endpoint not found (HTTP 404). Make sure the worker is deployed at %s", c.serverAddr)}
			case http.StatusServiceUnavailable:
				return fmt.Errorf("worker is unavailable (HTTP 503) — it may be starting up")
			default:
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(body))
			}
		}
		// DNS / network level failure
		if isNetworkUnreachable(err) {
			return fmt.Errorf("cannot reach worker at %s — check your internet connection or worker URL", c.serverAddr)
		}
		return fmt.Errorf("dial failed: %v", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	log.Printf("✓  [%s] Connected → https://%s/t/%s/", c.tunnelID, c.serverAddr, c.tunnelID)
	return nil
}

func (c *Client) listen() {
	defer func() {
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.stopChan:
			return
		default:
		}

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			// Suppress noisy "use of closed" errors on intentional stop
			select {
			case <-c.stopChan:
			default:
				log.Printf("⚠  [%s] Read error: %v", c.tunnelID, err)
			}
			return
		}

		var req ProxyRequest
		if err := json.Unmarshal(message, &req); err != nil {
			log.Printf("⚠  [%s] Bad message: %v", c.tunnelID, err)
			continue
		}

		if req.Type == "req" {
			go c.handleRequest(req)
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

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		log.Printf("⚠  [%s] Local server error: %v", c.tunnelID, err)
		c.sendResponse(ProxyResponse{Type: "res", ID: req.ID, Status: 502})
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

func (c *Client) sendResponse(res ProxyResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return
	}
	data, _ := json.Marshal(res)
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("write error: %v", err)
	}
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
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return fmt.Errorf("local server at %s is not reachable. Make sure your app is running", c.localAddr)
	}
	conn.Close()
	return nil
}

func (c *Client) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return
	}
	c.stopped = true
	close(c.stopChan)

	if c.conn != nil {
		c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
		c.conn = nil
	}
}

// fatalError is a non-retryable error (e.g. auth failure).
type fatalError struct{ msg string }

func (e *fatalError) Error() string { return e.msg }

func isFatal(err error) bool {
	_, ok := err.(*fatalError)
	return ok
}

func isNetworkUnreachable(err error) bool {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	if _, ok := err.(*net.DNSError); ok {
		return true
	}
	return false
}
