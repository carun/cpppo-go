package cpppo

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestCIPStatusToError(t *testing.T) {
	tests := []struct {
		status   byte
		hasError bool
	}{
		{0x00, false}, // Success
		{0x01, true},  // Connection failure
		{0x02, true},  // Resource unavailable
		{0xFF, true},  // General Error
	}

	for _, tc := range tests {
		err := CIPStatusToError(tc.status)
		if tc.hasError && err == nil {
			t.Errorf("Status %#x should return an error", tc.status)
		}
		if !tc.hasError && err != nil {
			t.Errorf("Status %#x should not return an error, got %v", tc.status, err)
		}
	}
}

func TestBuildCIPPath(t *testing.T) {
	tests := []struct {
		tagName      string
		expectedPath []byte
	}{
		{"", []byte{}},
		{"Tag1", []byte{CIPPathTypeSymbolic, 4, 'T', 'a', 'g', '1'}},
		{"LongTagName", []byte{CIPPathTypeSymbolic, 11, 'L', 'o', 'n', 'g', 'T', 'a', 'g', 'N', 'a', 'm', 'e', 0}}, // With padding
	}

	for _, tc := range tests {
		path := BuildCIPPath(tc.tagName)
		if !bytes.Equal(path, tc.expectedPath) {
			t.Errorf("For tag '%s', expected path %v, got %v", tc.tagName, tc.expectedPath, path)
		}
	}
}

func TestBuildCIPReadRequest(t *testing.T) {
	// Test with a simple tag
	tag := "Counter"
	elements := uint16(1)

	request := BuildCIPReadRequest(tag, elements)

	// Verify the request structure
	if request[0] != CIPServiceReadTag {
		t.Errorf("Expected service code %#x, got %#x", CIPServiceReadTag, request[0])
	}

	// Path size in 16-bit words (request[1])
	expectedPathSize := (len(tag) + 2) / 2 // +2 for symbolic segment byte and length byte
	if len(tag)%2 != 0 {
		expectedPathSize++ // Add padding
	}

	if request[1] != byte(expectedPathSize) {
		t.Errorf("Expected path size %d, got %d", expectedPathSize, request[1])
	}

	// Verify elements count at the end
	elementsBytes := request[len(request)-2:]
	gotElements := binary.LittleEndian.Uint16(elementsBytes)
	if gotElements != elements {
		t.Errorf("Expected elements %d, got %d", elements, gotElements)
	}
}

func TestBuildCIPWriteRequest(t *testing.T) {
	// Test with a DINT (int32) tag
	tag := "Counter"
	dataType := byte(CIPDataTypeDINT)
	value := int32(42)

	// Convert value to bytes
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(value))

	request := BuildCIPWriteRequest(tag, dataType, data)

	// Verify the request structure
	if request[0] != CIPServiceWriteTag {
		t.Errorf("Expected service code %#x, got %#x", CIPServiceWriteTag, request[0])
	}

	// Calculate path length
	pathLength := 2 // Symbolic segment byte and length byte
	pathLength += len(tag)
	if len(tag)%2 != 0 {
		pathLength++ // Add padding
	}

	// The data type byte should be at the end of the path
	if request[pathLength] != dataType {
		t.Errorf("Expected data type %#x, got %#x", dataType, request[pathLength])
	}

	// Check elements count (should be 1)
	if request[pathLength+1] != 1 {
		t.Errorf("Expected elements 1, got %d", request[pathLength+1])
	}

	// Verify data (should start after the data type and elements count)
	valueStart := pathLength + 2
	valueEnd := valueStart + len(data)
	if valueEnd > len(request) {
		t.Fatalf("Request too short: expected at least %d bytes, got %d", valueEnd, len(request))
	}

	valueBytes := request[valueStart:valueEnd]
	if !bytes.Equal(valueBytes, data) {
		t.Errorf("Expected value bytes %v, got %v", data, valueBytes)
	}
}
func TestParseCIPResponse(t *testing.T) {
	// Test successful response
	successResp := []byte{0x8A, 0x00, 'D', 'A', 'T', 'A'} // Service 0x0A with success bit (0x80) set, status 0, data "DATA"
	data, err := ParseCIPResponse(successResp)
	if err != nil {
		t.Errorf("Failed to parse successful response: %v", err)
	}
	if !bytes.Equal(data, []byte{'D', 'A', 'T', 'A'}) {
		t.Errorf("Expected data 'DATA', got %v", data)
	}

	// Test error response
	errorResp := []byte{0x8A, 0x01, 0x02} // Service 0x0A with success bit set, status 1 (error), extended status 2
	_, err = ParseCIPResponse(errorResp)
	if err == nil {
		t.Error("Expected error for error response, got nil")
	}

	// Test invalid response
	invalidResp := []byte{0x0A, 0x00} // Service bit not set (not a response)
	_, err = ParseCIPResponse(invalidResp)
	if err == nil {
		t.Error("Expected error for invalid response, got nil")
	}
}

func TestParseCIPReadResponse(t *testing.T) {
	// Test DINT response
	dintResp := []byte{0xCC, 0x00, CIPDataTypeDINT, 0x01, 42, 0, 0, 0} // Success, DINT, 1 element, value 42
	value, err := ParseCIPReadResponse(dintResp, CIPDataTypeDINT)
	if err != nil {
		t.Errorf("Failed to parse DINT response: %v", err)
	}

	intValue, ok := value.(int32)
	if !ok {
		t.Errorf("Expected int32 value, got %T", value)
	} else if intValue != 42 {
		t.Errorf("Expected value 42, got %d", intValue)
	}

	// Test BOOL response
	boolResp := []byte{0xCC, 0x00, CIPDataTypeBOOL, 0x01, 1} // Success, BOOL, 1 element, value true
	value, err = ParseCIPReadResponse(boolResp, CIPDataTypeBOOL)
	if err != nil {
		t.Errorf("Failed to parse BOOL response: %v", err)
	}

	boolValue, ok := value.(bool)
	if !ok {
		t.Errorf("Expected bool value, got %T", value)
	} else if !boolValue {
		t.Errorf("Expected value true, got %v", boolValue)
	}

	// Test data type mismatch
	mismatchResp := []byte{0xCC, 0x00, CIPDataTypeREAL, 0x01, 0, 0, 0, 0} // Success, REAL, but expected DINT
	_, err = ParseCIPReadResponse(mismatchResp, CIPDataTypeDINT)
	if err == nil {
		t.Error("Expected error for data type mismatch, got nil")
	}
}
