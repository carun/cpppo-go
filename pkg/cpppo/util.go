package cpppo

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"time"
)

// Common utility functions for the CPPPO library

// ExponentialBackoff implements an exponential backoff retry mechanism
func ExponentialBackoff(operation func() error, initialDelay, maxDelay time.Duration, maxRetries int) error {
	var err error
	delay := initialDelay

	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}

		// Check if this is a network error that we should retry
		if netErr, ok := err.(net.Error); ok && (netErr.Timeout() || isConnectionError(netErr)) {
			time.Sleep(delay)
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}

		// Not a temporary network error, so don't retry
		return err
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, err)
}

// Check if the error is a connection error that should be retried
func isConnectionError(err error) bool {
	return strings.Contains(err.Error(), "connection") ||
		strings.Contains(err.Error(), "reset") ||
		strings.Contains(err.Error(), "broken pipe")
}

// FormatTagName ensures a tag name is properly formatted for CIP
func FormatTagName(program, tag string) string {
	if program == "" {
		return tag
	}
	return fmt.Sprintf("%s.%s", program, tag)
}

// ParseIPAddress parses an IP address and ensures it has a port
func ParseIPAddress(address string, defaultPort int) string {
	// Add default port if not specified
	if _, _, err := net.SplitHostPort(address); err != nil {
		return fmt.Sprintf("%s:%d", address, defaultPort)
	}
	return address
}

// EncodeBool encodes a boolean value for CIP
func EncodeBool(value bool) []byte {
	if value {
		return []byte{1}
	}
	return []byte{0}
}

// EncodeInt16 encodes an int16 value for CIP
func EncodeInt16(value int16) []byte {
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, uint16(value))
	return data
}

// EncodeInt32 encodes an int32 value for CIP
func EncodeInt32(value int32) []byte {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(value))
	return data
}

// EncodeFloat32 encodes a float32 value for CIP
func EncodeFloat32(value float32) []byte {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, math.Float32bits(value))
	return data
}

// DecodeBool decodes a CIP boolean value
func DecodeBool(data []byte) (bool, error) {
	if len(data) < 1 {
		return false, errors.New("not enough data to decode bool")
	}
	return data[0] != 0, nil
}

// DecodeInt16 decodes a CIP int16 value
func DecodeInt16(data []byte) (int16, error) {
	if len(data) < 2 {
		return 0, errors.New("not enough data to decode int16")
	}
	return int16(binary.LittleEndian.Uint16(data)), nil
}

// DecodeInt32 decodes a CIP int32 value
func DecodeInt32(data []byte) (int32, error) {
	if len(data) < 4 {
		return 0, errors.New("not enough data to decode int32")
	}
	return int32(binary.LittleEndian.Uint32(data)), nil
}

// DecodeFloat32 decodes a CIP float32 value
func DecodeFloat32(data []byte) (float32, error) {
	if len(data) < 4 {
		return 0, errors.New("not enough data to decode float32")
	}
	bits := binary.LittleEndian.Uint32(data)
	return math.Float32frombits(bits), nil
}
