package fanuc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

// LogType represents different types of Fanuc logs
type LogType string

const (
	// Common log types in Fanuc controllers
	LogTypeAlarm   LogType = "ALARM"   // Alarm log
	LogTypeError   LogType = "ERROR"   // Error log
	LogTypeEvent   LogType = "EVENT"   // Event log
	LogTypeSystem  LogType = "SYSTEM"  // System log
	LogTypeComm    LogType = "COMM"    // Communication log
	LogTypeMotion  LogType = "MOTION"  // Motion log
	LogTypeProgram LogType = "PROGRAM" // Program execution log
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	// Log levels
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
	LogLevelFatal
)

// LogEntry represents a single log entry from the Fanuc controller
type LogEntry struct {
	Timestamp time.Time // When the log entry was generated
	Type      LogType   // Type of log
	Level     LogLevel  // Severity level
	Message   string    // Log message
	Code      string    // Error/alarm code (if applicable)
	Details   string    // Additional details
}

// LogReader reads logs from a Fanuc controller
type LogReader struct {
	address string        // Controller address (IP:port)
	timeout time.Duration // Connection timeout
	conn    net.Conn      // Network connection
	mutex   sync.Mutex    // Mutex for thread safety
	// connectOnce sync.Once     // Ensure single connection attempt
	connected bool // Connection status
}

// NewLogReader creates a new Fanuc log reader
func NewLogReader(address string, timeout time.Duration) *LogReader {
	// Add default port if not specified
	if _, _, err := net.SplitHostPort(address); err != nil {
		// Use port 18735 which is commonly used for Fanuc logs
		address = fmt.Sprintf("%s:18735", address)
	}

	return &LogReader{
		address: address,
		timeout: timeout,
	}
}

// Connect establishes a connection to the Fanuc controller
func (lr *LogReader) Connect() error {
	lr.mutex.Lock()
	defer lr.mutex.Unlock()

	if lr.connected {
		return nil // Already connected
	}

	var err error
	lr.conn, err = net.DialTimeout("tcp", lr.address, lr.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to log server: %w", err)
	}

	// Send authentication if required (depends on controller configuration)
	// This is a simplified example - actual authentication might vary
	auth := []byte("CONNECT_LOG_READER\n")
	if _, err := lr.conn.Write(auth); err != nil {
		lr.conn.Close()
		return fmt.Errorf("failed to send authentication: %w", err)
	}

	// Read authentication response
	response := make([]byte, 128)
	if err := lr.conn.SetReadDeadline(time.Now().Add(lr.timeout)); err != nil {
		lr.conn.Close()
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	n, err := lr.conn.Read(response)
	if err != nil {
		lr.conn.Close()
		return fmt.Errorf("failed to read authentication response: %w", err)
	}

	// Check for success response (simplified - actual format may vary)
	if !strings.Contains(string(response[:n]), "OK") {
		lr.conn.Close()
		return errors.New("authentication failed")
	}

	lr.connected = true
	return nil
}

// Close closes the connection to the Fanuc controller
func (lr *LogReader) Close() error {
	lr.mutex.Lock()
	defer lr.mutex.Unlock()

	if !lr.connected || lr.conn == nil {
		return nil
	}

	err := lr.conn.Close()
	lr.connected = false
	return err
}

// ReadLogs reads log entries from the Fanuc controller
// It returns a channel that will receive log entries
func (lr *LogReader) ReadLogs(ctx context.Context) (<-chan LogEntry, error) {
	// Ensure we're connected
	err := lr.Connect()
	if err != nil {
		return nil, err
	}

	logChan := make(chan LogEntry, 100) // Buffer for 100 log entries

	go func() {
		defer close(logChan)
		defer lr.Close()

		// Create a reader for the connection
		reader := bufio.NewReader(lr.conn)

		for {
			select {
			case <-ctx.Done():
				// Context canceled
				return
			default:
				// Read the next log entry
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						// Connection closed
						return
					}
					// Try to reconnect on error
					lr.reconnect()
					time.Sleep(1 * time.Second)
					continue
				}

				// Parse the log entry
				entry, err := lr.parseLogEntry(line)
				if err != nil {
					// Skip entries that can't be parsed
					continue
				}

				// Send the entry to the channel
				select {
				case logChan <- entry:
					// Entry sent successfully
				case <-ctx.Done():
					// Context canceled
					return
				}
			}
		}
	}()

	return logChan, nil
}

// reconnect attempts to reconnect to the log server
func (lr *LogReader) reconnect() {
	lr.mutex.Lock()
	defer lr.mutex.Unlock()

	if lr.conn != nil {
		lr.conn.Close()
		lr.connected = false
	}

	// Try to reconnect
	conn, err := net.DialTimeout("tcp", lr.address, lr.timeout)
	if err != nil {
		return
	}

	// Send authentication if required
	auth := []byte("CONNECT_LOG_READER\n")
	if _, err := conn.Write(auth); err != nil {
		conn.Close()
		return
	}

	// Read authentication response
	response := make([]byte, 128)
	if err := conn.SetReadDeadline(time.Now().Add(lr.timeout)); err != nil {
		conn.Close()
		return
	}

	n, err := conn.Read(response)
	if err != nil {
		conn.Close()
		return
	}

	// Check for success response
	if !strings.Contains(string(response[:n]), "OK") {
		conn.Close()
		return
	}

	lr.conn = conn
	lr.connected = true
}

// parseLogEntry parses a log entry from a string
func (lr *LogReader) parseLogEntry(line string) (LogEntry, error) {
	// Trim whitespace
	line = strings.TrimSpace(line)
	if line == "" {
		return LogEntry{}, errors.New("empty log entry")
	}

	// Parse the log entry
	// Format varies by controller and configuration
	// This is a simplified example - adjust for your controller

	// Example format: "[TIMESTAMP] [TYPE] [LEVEL] [CODE] MESSAGE"
	// e.g. "[2023-01-01 12:34:56] [ALARM] [ERROR] [SRVO-001] Servo error"

	entry := LogEntry{
		Timestamp: time.Now(), // Default to current time
		Type:      LogTypeSystem,
		Level:     LogLevelInfo,
		Message:   line, // Default to the entire line
	}

	// Try to parse timestamp
	timestampPattern := regexp.MustCompile(`\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]`)
	if match := timestampPattern.FindStringSubmatch(line); len(match) > 1 {
		if timestamp, err := time.Parse("2006-01-02 15:04:05", match[1]); err == nil {
			entry.Timestamp = timestamp
		}
	}

	// Try to parse log type
	typePattern := regexp.MustCompile(`\[(ALARM|ERROR|EVENT|SYSTEM|COMM|MOTION|PROGRAM)\]`)
	if match := typePattern.FindStringSubmatch(line); len(match) > 1 {
		entry.Type = LogType(match[1])
	}

	// Try to parse log level
	levelPattern := regexp.MustCompile(`\[(DEBUG|INFO|WARNING|ERROR|FATAL)\]`)
	if match := levelPattern.FindStringSubmatch(line); len(match) > 1 {
		switch match[1] {
		case "DEBUG":
			entry.Level = LogLevelDebug
		case "INFO":
			entry.Level = LogLevelInfo
		case "WARNING":
			entry.Level = LogLevelWarning
		case "ERROR":
			entry.Level = LogLevelError
		case "FATAL":
			entry.Level = LogLevelFatal
		}
	}

	// Try to parse error code
	codePattern := regexp.MustCompile(`\[([A-Z]+-\d+)\]`)
	if match := codePattern.FindStringSubmatch(line); len(match) > 1 {
		entry.Code = match[1]
	}

	// Extract message (everything after the last ])
	messagePattern := regexp.MustCompile(`\][^\]]*$`)
	if match := messagePattern.FindString(line); len(match) > 1 {
		entry.Message = strings.TrimSpace(match[1:])
	}

	return entry, nil
}

// FilterLogsByType filters logs by type
func (lr *LogReader) FilterLogsByType(ctx context.Context, logType LogType) (<-chan LogEntry, error) {
	logs, err := lr.ReadLogs(ctx)
	if err != nil {
		return nil, err
	}

	filteredLogs := make(chan LogEntry, 100)

	go func() {
		defer close(filteredLogs)

		for entry := range logs {
			if entry.Type == logType {
				select {
				case filteredLogs <- entry:
					// Entry sent successfully
				case <-ctx.Done():
					// Context canceled
					return
				}
			}
		}
	}()

	return filteredLogs, nil
}

// GetLatestAlarms gets the latest alarms from the controller
func (lr *LogReader) GetLatestAlarms(ctx context.Context, count int) ([]LogEntry, error) {
	// Request alarm history from the controller
	err := lr.Connect()
	if err != nil {
		return nil, err
	}

	// Send command to get alarm history
	cmd := fmt.Sprintf("GET_ALARM_HISTORY %d\n", count)
	if _, err := lr.conn.Write([]byte(cmd)); err != nil {
		return nil, fmt.Errorf("failed to send alarm history request: %w", err)
	}

	// Read response header
	reader := bufio.NewReader(lr.conn)
	header, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read alarm history header: %w", err)
	}

	// Parse header to get number of alarms
	header = strings.TrimSpace(header)
	var numAlarms int
	_, err = fmt.Sscanf(header, "ALARM_HISTORY %d", &numAlarms)
	if err != nil {
		return nil, fmt.Errorf("failed to parse alarm history header: %w", err)
	}

	// Read alarm entries
	alarms := make([]LogEntry, 0, numAlarms)
	for i := 0; i < numAlarms; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		entry, err := lr.parseLogEntry(line)
		if err != nil {
			continue
		}

		entry.Type = LogTypeAlarm
		alarms = append(alarms, entry)
	}

	return alarms, nil
}

// RemoteLogRequest represents a request for remote log monitoring
type RemoteLogRequest struct {
	Types []LogType // Log types to include
	Since time.Time // Only include logs since this time
	Regex string    // Regex pattern to filter logs
}

// StartRemoteLogMonitor starts remote monitoring of logs
func (lr *LogReader) StartRemoteLogMonitor(ctx context.Context, request RemoteLogRequest) (<-chan LogEntry, error) {
	err := lr.Connect()
	if err != nil {
		return nil, err
	}

	// Construct command to start remote monitoring
	// Format: START_MONITOR [TYPE1,TYPE2,...] [SINCE=timestamp] [REGEX=pattern]
	cmd := "START_MONITOR"

	if len(request.Types) > 0 {
		typeStrs := make([]string, len(request.Types))
		for i, t := range request.Types {
			typeStrs[i] = string(t)
		}
		cmd += " " + strings.Join(typeStrs, ",")
	}

	if !request.Since.IsZero() {
		cmd += fmt.Sprintf(" SINCE=%s", request.Since.Format("2006-01-02T15:04:05"))
	}

	if request.Regex != "" {
		cmd += fmt.Sprintf(" REGEX=%s", request.Regex)
	}

	cmd += "\n"

	// Send command
	if _, err := lr.conn.Write([]byte(cmd)); err != nil {
		return nil, fmt.Errorf("failed to send monitor request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(lr.conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read monitor response: %w", err)
	}

	response = strings.TrimSpace(response)
	if !strings.HasPrefix(response, "OK") {
		return nil, fmt.Errorf("monitor request failed: %s", response)
	}

	// Start reading logs
	return lr.ReadLogs(ctx)
}

// StopRemoteLogMonitor stops remote monitoring of logs
func (lr *LogReader) StopRemoteLogMonitor() error {
	lr.mutex.Lock()
	defer lr.mutex.Unlock()

	if !lr.connected || lr.conn == nil {
		return nil
	}

	// Send command to stop monitoring
	cmd := "STOP_MONITOR\n"
	if _, err := lr.conn.Write([]byte(cmd)); err != nil {
		return fmt.Errorf("failed to send stop monitor request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(lr.conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read stop monitor response: %w", err)
	}

	response = strings.TrimSpace(response)
	if !strings.HasPrefix(response, "OK") {
		return fmt.Errorf("stop monitor request failed: %s", response)
	}

	return nil
}
