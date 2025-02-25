package cpppo

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// CIP Service Codes
const (
	CIPServiceGetAttributeAll  = 0x01
	CIPServiceGetAttributeList = 0x03
	CIPServiceSetAttributeList = 0x04
	CIPServiceReset            = 0x05
	CIPServiceStart            = 0x06
	CIPServiceStop             = 0x07
	CIPServiceCreate           = 0x08
	CIPServiceDelete           = 0x09
	CIPServiceMultipleService  = 0x0A
	CIPServiceReadTag          = 0x4C
	CIPServiceWriteTag         = 0x4D
	CIPServiceReadModify       = 0x4E
)

// CIP Path Types
const (
	CIPPathTypeLogical  = 0x20
	CIPPathTypeSegment  = 0x28
	CIPPathTypeData     = 0x30
	CIPPathTypeSymbolic = 0x91
	CIPPathTypeANSI     = 0x92
)

// CIP Data Types
const (
	CIPDataTypeBOOL   = 0xC1
	CIPDataTypeSINT   = 0xC2
	CIPDataTypeINT    = 0xC3
	CIPDataTypeDINT   = 0xC4
	CIPDataTypeREAL   = 0xCA
	CIPDataTypeDWORD  = 0xD3
	CIPDataTypeSTRING = 0xD0
)

// CIPError represents a CIP error
type CIPError struct {
	Code        byte
	ExtendedMsg string
}

func (e CIPError) Error() string {
	return fmt.Sprintf("CIP Error: %#x - %s", e.Code, e.ExtendedMsg)
}

// CIPStatusToError converts a CIP status code to an error
func CIPStatusToError(status byte) error {
	if status == 0 {
		return nil
	}

	var msg string
	switch status {
	case 0x01:
		msg = "Connection failure"
	case 0x02:
		msg = "Resource unavailable"
	case 0x03:
		msg = "Invalid parameter value"
	case 0x04:
		msg = "Path segment error"
	case 0x05:
		msg = "Path destination unknown"
	case 0x06:
		msg = "Partial transfer"
	case 0x07:
		msg = "Connection lost"
	case 0x08:
		msg = "Service not supported"
	case 0x09:
		msg = "Invalid attribute value"
	case 0x0A:
		msg = "Attribute list error"
	case 0x0B:
		msg = "Already in requested mode/state"
	case 0x0C:
		msg = "Object state conflict"
	case 0x0D:
		msg = "Object already exists"
	case 0x0E:
		msg = "Attribute not settable"
	case 0x0F:
		msg = "Privilege violation"
	case 0x10:
		msg = "Device state conflict"
	case 0x11:
		msg = "Reply data too large"
	case 0x12:
		msg = "Fragmentation of a primitive value"
	case 0x13:
		msg = "Not enough data"
	case 0x14:
		msg = "Attribute not supported"
	case 0x15:
		msg = "Too much data"
	case 0x16:
		msg = "Object does not exist"
	case 0x17:
		msg = "Service fragmentation sequence not in progress"
	case 0x18:
		msg = "No stored attribute data"
	case 0x19:
		msg = "Store operation failure"
	case 0x1A:
		msg = "Routing failure, request packet too large"
	case 0x1B:
		msg = "Routing failure, response packet too large"
	case 0x1C:
		msg = "Missing attribute list entry data"
	case 0x1D:
		msg = "Invalid attribute value list"
	case 0x1E:
		msg = "Embedded service error"
	case 0x1F:
		msg = "Vendor specific error"
	case 0x20:
		msg = "Invalid parameter"
	case 0x21:
		msg = "Write-once value or medium already written"
	case 0x22:
		msg = "Invalid reply received"
	case 0x25:
		msg = "Key failure in path"
	case 0x26:
		msg = "Path size invalid"
	case 0x27:
		msg = "Unexpected attribute in list"
	case 0x28:
		msg = "Invalid member ID"
	case 0x29:
		msg = "Member not settable"
	case 0xFF:
		msg = "General Error"
	default:
		msg = "Unknown error"
	}

	return CIPError{
		Code:        status,
		ExtendedMsg: msg,
	}
}

// BuildCIPPath creates a CIP path from a tag name
func BuildCIPPath(tagName string) []byte {
	if tagName == "" {
		return []byte{}
	}

	// For symbolic tags, format is:
	// 0x91 (symbolic segment) + length of name + name
	length := len(tagName)

	// If length is odd, we need to pad with a zero byte
	padded := length%2 != 0

	// Create the path
	path := make([]byte, 2+length)
	path[0] = CIPPathTypeSymbolic
	path[1] = byte(length)

	// Copy the tag name
	copy(path[2:], []byte(tagName))

	// Add padding if needed
	if padded {
		path = append(path, 0)
	}

	return path
}

// BuildCIPReadRequest creates a CIP read request for a tag
func BuildCIPReadRequest(tagName string, elements uint16) []byte {
	path := BuildCIPPath(tagName)

	// Create the request
	request := make([]byte, 4+len(path))

	// Service code for Read Tag
	request[0] = CIPServiceReadTag

	// Request path size in words (16-bit chunks)
	request[1] = byte((len(path) + 1) / 2)

	// Copy the path
	copy(request[2:], path)

	// Number of elements to read
	binary.LittleEndian.PutUint16(request[len(request)-2:], elements)

	return request
}

// BuildCIPWriteRequest creates a CIP write request for a tag
func BuildCIPWriteRequest(tagName string, dataType byte, data []byte) []byte {
	path := BuildCIPPath(tagName)

	// Create the request
	request := make([]byte, 4+len(path)+len(data))

	// Service code for Write Tag
	request[0] = CIPServiceWriteTag

	// Request path size in words (16-bit chunks)
	request[1] = byte((len(path) + 1) / 2)

	// Copy the path
	copy(request[2:], path)

	// Data type
	request[2+len(path)] = dataType

	// Number of elements (always 1 for now)
	request[3+len(path)] = 1

	// Copy the data
	copy(request[4+len(path):], data)

	return request
}

// ParseCIPResponse parses a CIP response
func ParseCIPResponse(response []byte) ([]byte, error) {
	if len(response) < 2 {
		return nil, errors.New("response too short")
	}

	// Check if this is a response (bit 7 set in service code)
	if response[0]&0x80 == 0 {
		return nil, errors.New("not a response")
	}

	// Check the status
	status := response[1]
	if err := CIPStatusToError(status); err != nil {
		return nil, err
	}

	// Return the data portion (skipping service code, status, and extended status size)
	if len(response) <= 2 {
		return []byte{}, nil
	}

	return response[2:], nil
}

// ParseCIPReadResponse parses a CIP read response
func ParseCIPReadResponse(response []byte, dataType byte) (interface{}, error) {
	data, err := ParseCIPResponse(response)
	if err != nil {
		return nil, err
	}

	// Make sure we have at least the data type and length
	if len(data) < 2 {
		return nil, errors.New("response data too short")
	}

	// Check that the data type matches what we expect
	respDataType := data[0]
	if respDataType != dataType {
		return nil, fmt.Errorf("data type mismatch: expected %#x, got %#x", dataType, respDataType)
	}

	// Get the data based on the type
	switch dataType {
	case CIPDataTypeBOOL:
		if len(data) < 3 {
			return nil, errors.New("not enough data for BOOL")
		}
		return data[2] != 0, nil

	case CIPDataTypeSINT:
		if len(data) < 3 {
			return nil, errors.New("not enough data for SINT")
		}
		return int8(data[2]), nil

	case CIPDataTypeINT:
		if len(data) < 4 {
			return nil, errors.New("not enough data for INT")
		}
		return int16(binary.LittleEndian.Uint16(data[2:4])), nil

	case CIPDataTypeDINT:
		if len(data) < 6 {
			return nil, errors.New("not enough data for DINT")
		}
		return int32(binary.LittleEndian.Uint32(data[2:6])), nil

	case CIPDataTypeREAL:
		if len(data) < 6 {
			return nil, errors.New("not enough data for REAL")
		}
		bits := binary.LittleEndian.Uint32(data[2:6])
		return float32FromUint32(bits), nil

	case CIPDataTypeDWORD:
		if len(data) < 6 {
			return nil, errors.New("not enough data for DWORD")
		}
		return binary.LittleEndian.Uint32(data[2:6]), nil

	case CIPDataTypeSTRING:
		if len(data) < 4 {
			return nil, errors.New("not enough data for STRING header")
		}
		length := binary.LittleEndian.Uint16(data[2:4])
		if len(data) < int(4+length) {
			return nil, errors.New("string data truncated")
		}
		return string(data[4 : 4+length]), nil

	default:
		return data[2:], nil
	}
}

// Helper function to convert uint32 to float32 (IEEE 754)
func float32FromUint32(bits uint32) float32 {
	return float32FromUint32Go(bits)
}

// Go implementation for float32 conversion
func float32FromUint32Go(bits uint32) float32 {
	return float32(bits)
}
