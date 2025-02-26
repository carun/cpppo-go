package fanuc

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// mockLogServer creates a mock TCP server for testing the log reader
func mockLogServer(t *testing.T, handler func(net.Conn)) (string, func()) {
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

func TestNewLogReader(t *testing.T) {
	// Test with IP address only
	reader := NewLogReader("192.168.1.10", 5*time.Second)
	if reader.address != "192.168.1.10:18735" {
		t.Errorf("Expected address 192.168.1.10:18735, got %s", reader.address)
	}
	if reader.timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", reader.timeout)
	}

	// Test with IP:port
	reader = NewLogReader("192.168.1.10:1234", 5*time.Second)
	if reader.address != "192.168.1.10:1234" {
		t.Errorf("Expected address 192.168.1.10:1234, got %s", reader.address)
	}
}

func TestConnect(t *testing.T) {
	// Create a mock server that expects an authentication message
	addr, cleanup := mockLogServer(t, func(conn net.Conn) {
		// Expect authentication message
		reader := bufio.NewReader(conn)
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Errorf("Failed to read auth message: %v", err)
			return
		}

		if !strings.Contains(line, "CONNECT_LOG_READER") {
			t.Errorf("Expected auth message containing CONNECT_LOG_READER, got %s", line)
		}

		// Send OK response
		_, err = conn.Write([]byte("OK\n"))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
			return
		}
	})
	defer cleanup()

	// Create log reader
	reader := NewLogReader(addr, 1*time.Second)

	// Connect to the mock server
	err := reader.Connect()
	if err != nil {
		t.Errorf("Connect failed: %v", err)
	}

	if !reader.connected {
		t.Error("Expected connected to be true")
	}

	// Test connection reuse
	err = reader.Connect()
	if err != nil {
		t.Errorf("Connect (reuse) failed: %v", err)
	}

	// Cleanup
	reader.Close()
}

func TestClose(t *testing.T) {
	// Create a mock server
	addr, cleanup := mockLogServer(t, func(conn net.Conn) {
		// Accept connection and auth
		reader := bufio.NewReader(conn)
		line, _ := reader.ReadString('\n')
		if strings.Contains(line, "CONNECT_LOG_READER") {
			_, err := conn.Write([]byte("OK\n"))
			if err != nil {
				t.Errorf("Failed to write response: %v", err)
				return
			}
		}

		// Wait for connection to be closed
		buf := make([]byte, 1)
		_, err := conn.Read(buf) // This will return when conn is closed
		if err != nil {
			t.Errorf("Failed to read response %v", err)
		}
	})
	defer cleanup()

	// Create and connect log reader
	reader := NewLogReader(addr, 1*time.Second)
	err := reader.Connect()
	if err != nil {
		t.Errorf("Failed to connect: %v", err)
	}

	// Close the connection
	err = reader.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if reader.connected {
		t.Error("Expected connected to be false after close")
	}

	// Test closing when not connected
	reader = NewLogReader(addr, 1*time.Second)
	err = reader.Close()
	if err != nil {
		t.Errorf("Close when not connected failed: %v", err)
	}
}

func TestParseLogEntry(t *testing.T) {
	reader := NewLogReader("localhost", 1*time.Second)

	tests := []struct {
		line     string
		hasError bool
		logType  LogType
		level    LogLevel
		code     string
		hasMsg   bool
	}{
		{
			line:     "[2023-01-01 12:34:56] [ALARM] [ERROR] [SRVO-001] Servo error",
			hasError: false,
			logType:  LogTypeAlarm,
			level:    LogLevelError,
			code:     "SRVO-001",
			hasMsg:   true,
		},
		{
			line:     "[2023-01-01 12:34:56] [EVENT] [INFO] System started",
			hasError: false,
			logType:  LogTypeEvent,
			level:    LogLevelInfo,
			code:     "",
			hasMsg:   true,
		},
		{
			line:     "[SYSTEM] [DEBUG] Initializing subsystems",
			hasError: false,
			logType:  LogTypeSystem,
			level:    LogLevelDebug,
			code:     "",
			hasMsg:   true,
		},
		{
			line:     "", // Empty line
			hasError: true,
			logType:  "",
			level:    0,
			code:     "",
			hasMsg:   false,
		},
	}

	for _, tc := range tests {
		entry, err := reader.parseLogEntry(tc.line)

		if tc.hasError && err == nil {
			t.Errorf("Expected error for line %q, got nil", tc.line)
			continue
		}

		if !tc.hasError && err != nil {
			t.Errorf("Unexpected error for line %q: %v", tc.line, err)
			continue
		}

		if err != nil {
			continue
		}

		if entry.Type != tc.logType {
			t.Errorf("For line %q: expected type %s, got %s", tc.line, tc.logType, entry.Type)
		}

		if entry.Level != tc.level {
			t.Errorf("For line %q: expected level %d, got %d", tc.line, tc.level, entry.Level)
		}

		if entry.Code != tc.code {
			t.Errorf("For line %q: expected code %s, got %s", tc.line, tc.code, entry.Code)
		}

		if tc.hasMsg && entry.Message == "" {
			t.Errorf("For line %q: expected non-empty message, got empty", tc.line)
		}
	}
}

func TestGetLatestAlarms(t *testing.T) {
	// Create a mock server that returns alarm history
	addr, cleanup := mockLogServer(t, func(conn net.Conn) {
		// Accept connection and auth
		reader := bufio.NewReader(conn)
		line, _ := reader.ReadString('\n')
		if strings.Contains(line, "CONNECT_LOG_READER") {
			_, err := conn.Write([]byte("OK\n"))
			if err != nil {
				t.Errorf("Failed to write: %v", err)
			}
		}

		// Handle alarm history request
		line, _ = reader.ReadString('\n')
		if strings.Contains(line, "GET_ALARM_HISTORY") {
			// Send header and 2 alarms
			_, err := conn.Write([]byte("ALARM_HISTORY 2\n"))
			if err != nil {
				t.Errorf("Failed to write: %v", err)
			}
			_, err = conn.Write([]byte("[2023-01-01 12:34:56] [ALARM] [ERROR] [SRVO-001] Servo error\n"))
			if err != nil {
				t.Errorf("Failed to write: %v", err)
			}
			_, err = conn.Write([]byte("[2023-01-01 12:35:00] [ALARM] [ERROR] [SRVO-002] Motion error\n"))
			if err != nil {
				t.Errorf("Failed to write: %v", err)
			}
		}
	})
	defer cleanup()

	// Create log reader
	reader := NewLogReader(addr, 1*time.Second)

	// Get latest alarms
	ctx := context.Background()
	alarms, err := reader.GetLatestAlarms(ctx, 10)
	if err != nil {
		t.Errorf("GetLatestAlarms failed: %v", err)
	}

	if len(alarms) != 2 {
		t.Errorf("Expected 2 alarms, got %d", len(alarms))
	}

	if alarms[0].Type != LogTypeAlarm {
		t.Errorf("Expected alarm type %s, got %s", LogTypeAlarm, alarms[0].Type)
	}

	if alarms[0].Code != "SRVO-001" {
		t.Errorf("Expected alarm code SRVO-001, got %s", alarms[0].Code)
	}

	if alarms[1].Code != "SRVO-002" {
		t.Errorf("Expected alarm code SRVO-002, got %s", alarms[1].Code)
	}
}

func TestStartRemoteLogMonitor(t *testing.T) {
	// Create a mock server that streams logs
	addr, cleanup := mockLogServer(t, func(conn net.Conn) {
		// Accept connection and auth
		reader := bufio.NewReader(conn)
		line, _ := reader.ReadString('\n')
		if strings.Contains(line, "CONNECT_LOG_READER") {
			_, err := conn.Write([]byte("OK\n"))
			if err != nil {
				t.Errorf("Failed to write: %v", err)
			}
		}

		// Handle monitor request
		line, _ = reader.ReadString('\n')
		if strings.Contains(line, "START_MONITOR") {
			// Send OK and then some log entries
			_, err := conn.Write([]byte("OK\n"))
			if err != nil {
				t.Errorf("Failed to write: %v", err)
			}
			_, err = conn.Write([]byte("[2023-01-01 12:34:56] [EVENT] [INFO] System started\n"))
			if err != nil {
				t.Errorf("Failed to write: %v", err)
			}
			_, err = conn.Write([]byte("[2023-01-01 12:35:00] [ALARM] [ERROR] [SRVO-001] Servo error\n"))
			if err != nil {
				t.Errorf("Failed to write: %v", err)
			}

			// Wait a bit before sending another log
			time.Sleep(100 * time.Millisecond)
			_, err = conn.Write([]byte("[2023-01-01 12:35:10] [EVENT] [INFO] Operation completed\n"))
			if err != nil {
				t.Errorf("Failed to write: %v", err)
			}

			// Handle STOP_MONITOR
			line, _ = reader.ReadString('\n')
			if strings.Contains(line, "STOP_MONITOR") {
				_, err = conn.Write([]byte("OK\n"))
				if err != nil {
					t.Errorf("Failed to write: %v", err)
				}
			}
		}
	})
	defer cleanup()

	// Create log reader
	reader := NewLogReader(addr, 1*time.Second)

	// Start monitoring
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	request := RemoteLogRequest{
		Types: []LogType{LogTypeAlarm, LogTypeEvent},
		Since: time.Now().Add(-1 * time.Hour),
	}

	logs, err := reader.StartRemoteLogMonitor(ctx, request)
	if err != nil {
		t.Errorf("StartRemoteLogMonitor failed: %v", err)
	}

	// Read logs
	count := 0
	for entry := range logs {
		count++
		if entry.Type != LogTypeEvent && entry.Type != LogTypeAlarm {
			t.Errorf("Unexpected log type: %s", entry.Type)
		}
	}

	if count != 3 {
		t.Errorf("Expected 3 log entries, got %d", count)
	}

	// Test stopping the monitor
	err = reader.StopRemoteLogMonitor()
	if err != nil {
		t.Errorf("StopRemoteLogMonitor failed: %v", err)
	}
}

func TestFilterLogsByType(t *testing.T) {
	// Create a mock server that streams logs of different types
	addr, cleanup := mockLogServer(t, func(conn net.Conn) {
		// Accept connection and auth
		reader := bufio.NewReader(conn)
		line, _ := reader.ReadString('\n')
		if strings.Contains(line, "CONNECT_LOG_READER") {
			_, err := conn.Write([]byte("OK\n"))
			if err != nil {
				t.Errorf("Failed to write: %v", err)
			}
		}

		// Stream mixed log types
		_, err := conn.Write([]byte("[2023-01-01 12:34:56] [EVENT] [INFO] System started\n"))
		if err != nil {
			t.Errorf("Failed to write: %v", err)
		}
		_, err = conn.Write([]byte("[2023-01-01 12:35:00] [ALARM] [ERROR] [SRVO-001] Servo error\n"))
		if err != nil {
			t.Errorf("Failed to write: %v", err)
		}
		_, err = conn.Write([]byte("[2023-01-01 12:35:10] [EVENT] [INFO] Operation completed\n"))
		if err != nil {
			t.Errorf("Failed to write: %v", err)
		}
		_, err = conn.Write([]byte("[2023-01-01 12:35:20] [SYSTEM] [INFO] Heartbeat\n"))
		if err != nil {
			t.Errorf("Failed to write: %v", err)
		}
		_, err = conn.Write([]byte("[2023-01-01 12:35:30] [ALARM] [ERROR] [SRVO-002] Motion error\n"))
		if err != nil {
			t.Errorf("Failed to write: %v", err)
		}
		// Wait for context cancellation
		time.Sleep(1 * time.Second)
	})
	defer cleanup()

	// Create log reader
	reader := NewLogReader(addr, 1*time.Second)

	// Filter logs by type
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	alarmLogs, err := reader.FilterLogsByType(ctx, LogTypeAlarm)
	if err != nil {
		t.Errorf("FilterLogsByType failed: %v", err)
	}

	// Read alarm logs
	alarmCount := 0
	for entry := range alarmLogs {
		alarmCount++
		if entry.Type != LogTypeAlarm {
			t.Errorf("Expected log type %s, got %s", LogTypeAlarm, entry.Type)
		}
	}

	if alarmCount != 2 {
		t.Errorf("Expected 2 alarm entries, got %d", alarmCount)
	}
}
