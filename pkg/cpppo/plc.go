package cpppo

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// PLCClient provides a higher-level interface for PLC communication
type PLCClient struct {
	client *Client
}

// NewPLCClient creates a new PLC client
func NewPLCClient(address string, timeout time.Duration) (*PLCClient, error) {
	client, err := NewClient(address, timeout)
	if err != nil {
		return nil, err
	}

	if err := client.RegisterSession(); err != nil {
		client.Close()
		return nil, err
	}

	return &PLCClient{
		client: client,
	}, nil
}

// Close closes the PLC client
func (p *PLCClient) Close() error {
	return p.client.Close()
}

// ReadTag reads a tag from the PLC
func (p *PLCClient) ReadTag(tagName string, dataType byte) (interface{}, error) {
	// Build CIP read request
	request := BuildCIPReadRequest(tagName, 1)

	// Send request
	response, err := p.client.SendRRData(0, 10, request)
	if err != nil {
		return nil, err
	}

	// Parse response
	return ParseCIPReadResponse(response, dataType)
}

// WriteTag writes a value to a tag in the PLC
func (p *PLCClient) WriteTag(tagName string, dataType byte, value interface{}) error {
	var data []byte

	// Convert the value to the appropriate binary format based on data type
	switch dataType {
	case CIPDataTypeBOOL:
		boolValue, ok := value.(bool)
		if !ok {
			return fmt.Errorf("value is not a bool")
		}
		if boolValue {
			data = []byte{1}
		} else {
			data = []byte{0}
		}

	case CIPDataTypeSINT:
		intValue, ok := value.(int8)
		if !ok {
			return fmt.Errorf("value is not an int8")
		}
		data = []byte{byte(intValue)}

	case CIPDataTypeINT:
		intValue, ok := value.(int16)
		if !ok {
			return fmt.Errorf("value is not an int16")
		}
		data = make([]byte, 2)
		binary.LittleEndian.PutUint16(data, uint16(intValue))

	case CIPDataTypeDINT:
		intValue, ok := value.(int32)
		if !ok {
			return fmt.Errorf("value is not an int32")
		}
		data = make([]byte, 4)
		binary.LittleEndian.PutUint32(data, uint32(intValue))

	case CIPDataTypeREAL:
		floatValue, ok := value.(float32)
		if !ok {
			return fmt.Errorf("value is not a float32")
		}
		data = make([]byte, 4)
		binary.LittleEndian.PutUint32(data, math.Float32bits(floatValue))

	default:
		return fmt.Errorf("unsupported data type: %#x", dataType)
	}

	// Build CIP write request
	request := BuildCIPWriteRequest(tagName, dataType, data)

	// Send request
	response, err := p.client.SendRRData(0, 10, request)
	if err != nil {
		return err
	}

	// Parse response to check for errors
	_, err = ParseCIPResponse(response)
	return err
}
