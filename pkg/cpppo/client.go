package cpppo

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Constants for EtherNet/IP protocol
const (
	EIPCommandNOP             = 0x0000
	EIPCommandListIdentity    = 0x0063
	EIPCommandListInterfaces  = 0x0064
	EIPCommandRegisterSession = 0x0065
	EIPCommandUnregister      = 0x0066
	EIPCommandSendRRData      = 0x006F
	EIPCommandSendUnitData    = 0x0070
	EIPCommandIndicateStatus  = 0x0072
	EIPCommandCancel          = 0x0073

	EIPDefaultPort = 44818
)

// EIPHeader represents the EtherNet/IP encapsulation header
type EIPHeader struct {
	Command       uint16
	Length        uint16
	SessionHandle uint32
	Status        uint32
	SenderContext [8]byte
	Options       uint32
}

// Client represents a CPPPO client
type Client struct {
	conn          net.Conn
	sessionHandle uint32
	timeout       time.Duration
	mu            sync.Mutex
}

// NewClient creates a new CPPPO client
func NewClient(address string, timeout time.Duration) (*Client, error) {
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	// Add default port if not specified
	if _, _, err := net.SplitHostPort(address); err != nil {
		address = fmt.Sprintf("%s:%d", address, EIPDefaultPort)
	}

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &Client{
		conn:    conn,
		timeout: timeout,
	}, nil
}

// Close closes the connection
func (c *Client) Close() error {
	if c.sessionHandle != 0 {
		err := c.unregisterSession()
		if err != nil {
			return err
		}
	}
	return c.conn.Close()
}

// RegisterSession registers a new session with the EIP server
func (c *Client) RegisterSession() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sessionHandle != 0 {
		return nil // Already registered
	}

	header := EIPHeader{
		Command: EIPCommandRegisterSession,
		Length:  4, // Protocol version + options flag
	}

	// Buffer to hold the header and data
	data := make([]byte, 24+4) // Header (24) + data (4)
	
	// Write header to buffer
	binary.LittleEndian.PutUint16(data[0:2], header.Command)
	binary.LittleEndian.PutUint16(data[2:4], header.Length)
	binary.LittleEndian.PutUint32(data[4:8], header.SessionHandle)
	binary.LittleEndian.PutUint32(data[8:12], header.Status)
	copy(data[12:20], header.SenderContext[:])
	binary.LittleEndian.PutUint32(data[20:24], header.Options)

	// Protocol version (1.1) and options flag (0)
	binary.LittleEndian.PutUint16(data[24:26], 1)
	binary.LittleEndian.PutUint16(data[26:28], 0)

	// Set deadline for write
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send register session request
	if _, err := c.conn.Write(data); err != nil {
		return fmt.Errorf("failed to send register session request: %w", err)
	}

	// Set deadline for read
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read response
	respHeader := make([]byte, 28) // Header (24) + protocol version and flags (4)
	if _, err := io.ReadFull(c.conn, respHeader); err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response header
	respCmd := binary.LittleEndian.Uint16(respHeader[0:2])
	respLen := binary.LittleEndian.Uint16(respHeader[2:4])
	respSessionHandle := binary.LittleEndian.Uint32(respHeader[4:8])
	respStatus := binary.LittleEndian.Uint32(respHeader[8:12])

	if respCmd != EIPCommandRegisterSession {
		return fmt.Errorf("unexpected response command: %d", respCmd)
	}

	if respStatus != 0 {
		return fmt.Errorf("registration failed with status: %d", respStatus)
	}

	if respLen != 4 {
		return fmt.Errorf("unexpected response length: %d", respLen)
	}

	c.sessionHandle = respSessionHandle
	return nil
}

// unregisterSession unregisters the session with the EIP server
func (c *Client) unregisterSession() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sessionHandle == 0 {
		return nil // Not registered
	}

	header := EIPHeader{
		Command:       EIPCommandUnregister,
		Length:        0,
		SessionHandle: c.sessionHandle,
	}

	// Buffer to hold the header
	data := make([]byte, 24)
	
	// Write header to buffer
	binary.LittleEndian.PutUint16(data[0:2], header.Command)
	binary.LittleEndian.PutUint16(data[2:4], header.Length)
	binary.LittleEndian.PutUint32(data[4:8], header.SessionHandle)
	binary.LittleEndian.PutUint32(data[8:12], header.Status)
	copy(data[12:20], header.SenderContext[:])
	binary.LittleEndian.PutUint32(data[20:24], header.Options)

	// Set deadline for write
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send unregister session request
	if _, err := c.conn.Write(data); err != nil {
		return fmt.Errorf("failed to send unregister session request: %w", err)
	}

	c.sessionHandle = 0
	return nil
}

// ListIdentity sends a List Identity request and returns the response
func (c *Client) ListIdentity() ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	header := EIPHeader{
		Command: EIPCommandListIdentity,
		Length:  0,
	}

	// Buffer to hold the header
	data := make([]byte, 24)
	
	// Write header to buffer
	binary.LittleEndian.PutUint16(data[0:2], header.Command)
	binary.LittleEndian.PutUint16(data[2:4], header.Length)
	binary.LittleEndian.PutUint32(data[4:8], header.SessionHandle)
	binary.LittleEndian.PutUint32(data[8:12], header.Status)
	copy(data[12:20], header.SenderContext[:])
	binary.LittleEndian.PutUint32(data[20:24], header.Options)

	// Set deadline for write
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send list identity request
	if _, err := c.conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to send list identity request: %w", err)
	}

	// Set deadline for read
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read response header
	respHeader := make([]byte, 24)
	if _, err := io.ReadFull(c.conn, respHeader); err != nil {
		return nil, fmt.Errorf("failed to read response header: %w", err)
	}

	// Parse response header
	respCmd := binary.LittleEndian.Uint16(respHeader[0:2])
	respLen := binary.LittleEndian.Uint16(respHeader[2:4])
	respStatus := binary.LittleEndian.Uint32(respHeader[8:12])

	if respCmd != EIPCommandListIdentity {
		return nil, fmt.Errorf("unexpected response command: %d", respCmd)
	}

	if respStatus != 0 {
		return nil, fmt.Errorf("list identity failed with status: %d", respStatus)
	}

	// Read response data
	respData := make([]byte, respLen)
	if _, err := io.ReadFull(c.conn, respData); err != nil {
		return nil, fmt.Errorf("failed to read response data: %w", err)
	}

	return respData, nil
}

// SendRRData sends a Send RR Data request and returns the response
func (c *Client) SendRRData(interfaceHandle uint32, timeout uint16, data []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sessionHandle == 0 {
		return nil, errors.New("session not registered")
	}

	// Calculate the length of the data
	dataLen := len(data)
	
	// Total data length = interface handle (4) + timeout (2) + data
	totalLen := 6 + dataLen

	header := EIPHeader{
		Command:       EIPCommandSendRRData,
		Length:        uint16(totalLen),
		SessionHandle: c.sessionHandle,
	}

	// Buffer to hold the header and data
	buffer := make([]byte, 24+totalLen)
	
	// Write header to buffer
	binary.LittleEndian.PutUint16(buffer[0:2], header.Command)
	binary.LittleEndian.PutUint16(buffer[2:4], header.Length)
	binary.LittleEndian.PutUint32(buffer[4:8], header.SessionHandle)
	binary.LittleEndian.PutUint32(buffer[8:12], header.Status)
	copy(buffer[12:20], header.SenderContext[:])
	binary.LittleEndian.PutUint32(buffer[20:24], header.Options)

	// Write interface handle and timeout
	binary.LittleEndian.PutUint32(buffer[24:28], interfaceHandle)
	binary.LittleEndian.PutUint16(buffer[28:30], timeout)

	// Copy data
	copy(buffer[30:], data)

	// Set deadline for write
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send request
	if _, err := c.conn.Write(buffer); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Set deadline for read
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read response header
	respHeader := make([]byte, 24)
	if _, err := io.ReadFull(c.conn, respHeader); err != nil {
		return nil, fmt.Errorf("failed to read response header: %w", err)
	}

	// Parse response header
	respCmd := binary.LittleEndian.Uint16(respHeader[0:2])
	respLen := binary.LittleEndian.Uint16(respHeader[2:4])
	respStatus := binary.LittleEndian.Uint32(respHeader[8:12])

	if respCmd != EIPCommandSendRRData {
		return nil, fmt.Errorf("unexpected response command: %d", respCmd)
	}

	if respStatus != 0 {
		return nil, fmt.Errorf("request failed with status: %d", respStatus)
	}

	// Read interface handle and timeout
	respData := make([]byte, 6)
	if _, err := io.ReadFull(c.conn, respData); err != nil {
		return nil, fmt.Errorf("failed to read interface handle and timeout: %w", err)
	}

	// Read response data
	respDataLen := int(respLen) - 6
	if respDataLen <= 0 {
		return []byte{}, nil
	}

	respPayload := make([]byte, respDataLen)
	if _, err := io.ReadFull(c.conn, respPayload); err != nil {
		return nil, fmt.Errorf("failed to read response data: %w", err)
	}

	return respPayload, nil
}

// SendUnitData sends a Send Unit Data request and returns the response
func (c *Client) SendUnitData(interfaceHandle uint32, timeout uint16, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sessionHandle == 0 {
		return errors.New("session not registered")
	}

	// Calculate the length of the data
	dataLen := len(data)
	
	// Total data length = interface handle (4) + timeout (2) + data
	totalLen := 6 + dataLen

	header := EIPHeader{
		Command:       EIPCommandSendUnitData,
		Length:        uint16(totalLen),
		SessionHandle: c.sessionHandle,
	}

	// Buffer to hold the header and data
	buffer := make([]byte, 24+totalLen)
	
	// Write header to buffer
	binary.LittleEndian.PutUint16(buffer[0:2], header.Command)
	binary.LittleEndian.PutUint16(buffer[2:4], header.Length)
	binary.LittleEndian.PutUint32(buffer[4:8], header.SessionHandle)
	binary.LittleEndian.PutUint32(buffer[8:12], header.Status)
	copy(buffer[12:20], header.SenderContext[:])
	binary.LittleEndian.PutUint32(buffer[20:24], header.Options)

	// Write interface handle and timeout
	binary.LittleEndian.PutUint32(buffer[24:28], interfaceHandle)
	binary.LittleEndian.PutUint16(buffer[28:30], timeout)

	// Copy data
	copy(buffer[30:], data)

	// Set deadline for write
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send request
	if _, err := c.conn.Write(buffer); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	return nil
}
