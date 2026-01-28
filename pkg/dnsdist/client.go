package dnsdist

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Client represents a connection to a dnsdist control socket
type Client struct {
	address string
	apiKey  string
	conn    net.Conn
	reader  *bufio.Reader
	timeout time.Duration
}

// NewClient creates a new dnsdist client
func NewClient(address, apiKey string) *Client {
	return &Client{
		address: address,
		apiKey:  apiKey,
		timeout: 10 * time.Second,
	}
}

// SetTimeout sets the connection timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// Connect establishes a connection to the dnsdist control socket
func (c *Client) Connect() error {
	conn, err := net.DialTimeout("tcp", c.address, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.address, err)
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)

	// Perform authentication if API key is provided
	if c.apiKey != "" {
		if err := c.authenticate(); err != nil {
			c.conn.Close()
			c.conn = nil
			c.reader = nil
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	return nil
}

// authenticate performs HMAC-based authentication with the dnsdist server
func (c *Client) authenticate() error {
	// Set read deadline for authentication
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	
	// Read the challenge from the server
	challenge, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read challenge: %w", err)
	}

	challenge = strings.TrimSpace(challenge)
	
	// Compute HMAC-SHA256 of the challenge
	h := hmac.New(sha256.New, []byte(c.apiKey))
	h.Write([]byte(challenge))
	response := hex.EncodeToString(h.Sum(nil))

	// Send the response
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}

	_, err = fmt.Fprintf(c.conn, "%s\n", response)
	if err != nil {
		return fmt.Errorf("failed to send auth response: %w", err)
	}

	// Reset deadlines after authentication
	c.conn.SetReadDeadline(time.Time{})
	c.conn.SetWriteDeadline(time.Time{})

	return nil
}

// ExecuteCommand sends a command to the dnsdist server and returns the response
func (c *Client) ExecuteCommand(command string) (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("not connected")
	}

	// Set write deadline
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return "", err
	}

	// Send the command
	_, err := fmt.Fprintf(c.conn, "%s\n", command)
	if err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return "", err
	}

	// Read the response
	var response strings.Builder
	
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to read response: %w", err)
		}
		
		response.WriteString(line)
		
		// Check if we've reached the end of the response
		// dnsdist typically ends responses with a prompt or specific marker
		if strings.HasPrefix(line, "> ") || strings.TrimSpace(line) == "" {
			break
		}
	}

	// Reset deadlines
	c.conn.SetReadDeadline(time.Time{})
	c.conn.SetWriteDeadline(time.Time{})

	return strings.TrimSpace(response.String()), nil
}

// Close closes the connection to the dnsdist server
func (c *Client) Close() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.reader = nil
		return err
	}
	return nil
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	return c.conn != nil
}
