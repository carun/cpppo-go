package cpppo

import (
	"net"
	"testing"
	"time"
)

// setupMockServer creates a mock TCP server for testing
func setupMockServer(t *testing.T, handler func(conn net.Conn)) (string, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		handler(conn)
	}()

	return listener.Addr().String(), func() {
		listener.Close()
	}
}

func TestNewClient(t *testing.T) {
	// Test with invalid address
	_, err := NewClient("invalid:port", 1*time.Second)
	if err == nil {
		t.Error("Expected error with invalid address, got nil")
	}

	// Test with valid server
	addr, cleanup := setupMockServer(t, func(conn net.Conn) {
		// Just accept connection and do nothing
	})
	defer cleanup()

	client, err := NewClient(addr, 1*time.Second)
	if err != nil {
		t.Errorf("Failed to create client: %v", err)
	}
	defer client.Close()

	if client.conn == nil {
		t.Error("Client connection is nil")
	}
}

func TestRegisterSession(t *testing.T) {
	// Mock server that handles the register session request
	addr, cleanup := setupMockServer(t, func(conn net.Conn) {
		// Read request header
		buf := make([]byte, 28)
		n, err := conn.Read(buf)
		if err != nil || n < 28 {
			t.Errorf("Failed to read request: %v", err)
			return
		}

		// Verify it's a register session request
		if buf[0] != byte(EIPCommandRegisterSession&0xFF) || buf[1] != byte(EIPCommandRegisterSession>>8) {
			t.Errorf("Unexpected command: %02x%02x", buf[1], buf[0])
			return
		}

		// Send back a successful response
		resp := make([]byte, 28)
		resp[0] = byte(EIPCommandRegisterSession & 0xFF)
		resp[1] = byte(EIPCommandRegisterSession >> 8)
		resp[2] = 4 // Length (low byte)
		resp[3] = 0 // Length (high byte)
		resp[4] = 1 // Session handle (low byte)
		resp[5] = 0
		resp[6] = 0
		resp[7] = 0 // Session handle (high byte)
		// Status is 0 (success)
		// Version is 1
		resp[24] = 1
		resp[25] = 0
		resp[26] = 0
		resp[27] = 0

		_, err = conn.Write(resp)
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
			return
		}
	})
	defer cleanup()

	client, err := NewClient(addr, 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.RegisterSession()
	if err != nil {
		t.Errorf("Failed to register session: %v", err)
	}

	if client.sessionHandle != 1 {
		t.Errorf("Expected session handle 1, got %d", client.sessionHandle)
	}
}

func TestListIdentity(t *testing.T) {
	// Mock server that handles the list identity request
	addr, cleanup := setupMockServer(t, func(conn net.Conn) {
		// Read request header
		buf := make([]byte, 24)
		n, err := conn.Read(buf)
		if err != nil || n < 24 {
			t.Errorf("Failed to read request: %v", err)
			return
		}

		// Verify it's a list identity request
		if buf[0] != byte(EIPCommandListIdentity&0xFF) || buf[1] != byte(EIPCommandListIdentity>>8) {
			t.Errorf("Unexpected command: %02x%02x", buf[1], buf[0])
			return
		}

		// Send back a minimal response
		resp := make([]byte, 24+8) // Header + some identity data
		resp[0] = byte(EIPCommandListIdentity & 0xFF)
		resp[1] = byte(EIPCommandListIdentity >> 8)
		resp[2] = 8 // Length (low byte)
		resp[3] = 0 // Length (high byte)
		// Status is 0 (success)

		// Simple identity data (8 bytes)
		resp[24] = 1 // Item count
		resp[25] = 0
		resp[26] = 0   // Type ID (low byte)
		resp[27] = 0   // Type ID (high byte)
		resp[28] = 4   // Length (low byte)
		resp[29] = 0   // Length (high byte)
		resp[30] = 'T' // Some data
		resp[31] = 'E'

		_, err = conn.Write(resp)
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
			return
		}
	})
	defer cleanup()

	client, err := NewClient(addr, 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	data, err := client.ListIdentity()
	if err != nil {
		t.Errorf("Failed to list identity: %v", err)
	}

	if len(data) != 8 {
		t.Errorf("Expected 8 bytes of identity data, got %d", len(data))
	}
}

func TestSendRRData(t *testing.T) {
	// Mock server that handles the send RR data request
	addr, cleanup := setupMockServer(t, func(conn net.Conn) {
		// Read request header
		buf := make([]byte, 30)
		n, err := conn.Read(buf)
		if err != nil || n < 30 {
			t.Errorf("Failed to read request: %v", err)
			return
		}

		// Register session first
		if buf[0] == byte(EIPCommandRegisterSession&0xFF) && buf[1] == byte(EIPCommandRegisterSession>>8) {
			// Send back a successful response
			resp := make([]byte, 28)
			resp[0] = byte(EIPCommandRegisterSession & 0xFF)
			resp[1] = byte(EIPCommandRegisterSession >> 8)
			resp[2] = 4 // Length (low byte)
			resp[3] = 0 // Length (high byte)
			resp[4] = 1 // Session handle (low byte)
			resp[5] = 0
			resp[6] = 0
			resp[7] = 0 // Session handle (high byte)
			// Status is 0 (success)
			resp[24] = 1
			resp[25] = 0
			resp[26] = 0
			resp[27] = 0

			_, err := conn.Write(resp)
			if err != nil {
				t.Errorf("Failed to write: %v", err)
			}
			// Read the next request (SendRRData)
			buf = make([]byte, 40)
			n, err = conn.Read(buf)
			if err != nil || n < 30 {
				t.Errorf("Failed to read SendRRData request: %v", err)
				return
			}
		}

		// Verify it's a SendRRData request
		if buf[0] != byte(EIPCommandSendRRData&0xFF) || buf[1] != byte(EIPCommandSendRRData>>8) {
			t.Errorf("Unexpected command: %02x%02x", buf[1], buf[0])
			return
		}

		// Send back a response
		resp := make([]byte, 24+10) // Header + interface handle (4) + timeout (2) + data (4)
		resp[0] = byte(EIPCommandSendRRData & 0xFF)
		resp[1] = byte(EIPCommandSendRRData >> 8)
		resp[2] = 10 // Length (low byte)
		resp[3] = 0  // Length (high byte)
		resp[4] = 1  // Session handle (low byte)
		resp[5] = 0
		resp[6] = 0
		resp[7] = 0 // Session handle (high byte)
		// Status is 0 (success)

		// Interface handle (4) + timeout (2) + sample data (4)
		resp[24] = 0 // Interface handle
		resp[25] = 0
		resp[26] = 0
		resp[27] = 0
		resp[28] = 0 // Timeout
		resp[29] = 0
		resp[30] = 'D' // Sample data
		resp[31] = 'A'
		resp[32] = 'T'
		resp[33] = 'A'

		_, err = conn.Write(resp)
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
			return
		}
	})
	defer cleanup()

	client, err := NewClient(addr, 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Register session first
	err = client.RegisterSession()
	if err != nil {
		t.Fatalf("Failed to register session: %v", err)
	}

	// Send RR data
	data, err := client.SendRRData(0, 10, []byte("TEST"))
	if err != nil {
		t.Errorf("Failed to send RR data: %v", err)
	}

	if len(data) != 4 {
		t.Errorf("Expected 4 bytes of response data, got %d", len(data))
	}

	if string(data) != "DATA" {
		t.Errorf("Expected response data 'DATA', got '%s'", string(data))
	}
}
