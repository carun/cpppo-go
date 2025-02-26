package fanuc

import (
	"testing"

	"github.com/carun/cpppo-go/pkg/cpppo"
)

// mockPLCClient implements PLCClientInterface for testing
type mockPLCClient struct {
	readResponses  map[string]interface{}
	writeResponses map[string]error
	readCalls      map[string]int
	writeCalls     map[string]interface{}
	closed         bool
}

func newMockPLCClient() *mockPLCClient {
	return &mockPLCClient{
		readResponses:  make(map[string]interface{}),
		writeResponses: make(map[string]error),
		readCalls:      make(map[string]int),
		writeCalls:     make(map[string]interface{}),
	}
}

func (m *mockPLCClient) ReadTag(tagName string, dataType byte) (interface{}, error) {
	m.readCalls[tagName]++
	if response, ok := m.readResponses[tagName]; ok {
		return response, nil
	}
	return 0, nil
}

func (m *mockPLCClient) WriteTag(tagName string, dataType byte, value interface{}) error {
	m.writeCalls[tagName] = value
	if err, ok := m.writeResponses[tagName]; ok {
		return err
	}
	return nil
}

func (m *mockPLCClient) Close() error {
	m.closed = true
	return nil
}

// TestBuildRegisterTag tests the buildRegisterTag function
func TestBuildRegisterTag(t *testing.T) {
	tests := []struct {
		regType     RegisterType
		index       int
		expectedTag string
	}{
		{RegisterTypeR, 1, "R[1]"},
		{RegisterTypePR, 5, "PR[5]"},
		{RegisterTypeDI, 10, "DI[10]"},
		{RegisterTypeDO, 100, "DO[100]"},
		{RegisterTypeAI, 200, "AI[200]"},
		{RegisterTypeAO, 300, "AO[300]"},
		{RegisterTypeGI, 400, "GI[400]"},
		{RegisterTypeGO, 500, "GO[500]"},
		{RegisterTypeUR, 600, "UR[600]"},
		{RegisterTypeSR, 700, "SR[700]"},
		{RegisterTypeVR, 800, "VR[800]"},
	}

	for _, tc := range tests {
		tag := buildRegisterTag(tc.regType, tc.index)
		if tag != tc.expectedTag {
			t.Errorf("For register type %d, index %d: expected tag %s, got %s",
				tc.regType, tc.index, tc.expectedTag, tag)
		}
	}
}

// TestGetRegisterDataType tests the getRegisterDataType function
func TestGetRegisterDataType(t *testing.T) {
	tests := []struct {
		regType      RegisterType
		expectedType byte
	}{
		{RegisterTypeR, cpppo.CIPDataTypeREAL},
		{RegisterTypePR, cpppo.CIPDataTypeSTRING},
		{RegisterTypeDI, cpppo.CIPDataTypeBOOL},
		{RegisterTypeDO, cpppo.CIPDataTypeBOOL},
		{RegisterTypeAI, cpppo.CIPDataTypeREAL},
		{RegisterTypeAO, cpppo.CIPDataTypeREAL},
		{RegisterTypeGI, cpppo.CIPDataTypeBOOL},
		{RegisterTypeGO, cpppo.CIPDataTypeBOOL},
		{RegisterTypeUR, cpppo.CIPDataTypeSTRING},
		{RegisterTypeSR, cpppo.CIPDataTypeSTRING},
		{RegisterTypeVR, cpppo.CIPDataTypeSTRING},
	}

	for _, tc := range tests {
		dataType := getRegisterDataType(tc.regType)
		if dataType != tc.expectedType {
			t.Errorf("For register type %d: expected data type %d, got %d",
				tc.regType, tc.expectedType, dataType)
		}
	}
}

// TestReadRegister tests the ReadRegister function
func TestReadRegister(t *testing.T) {
	mock := newMockPLCClient()
	client := &FanucClient{PLCClient: mock}

	// Set up mock response for R register
	mock.readResponses["R[5]"] = float32(42.5)

	// Test reading an R register
	value, err := client.ReadRegister(RegisterTypeR, 5)
	if err != nil {
		t.Errorf("Failed to read R register: %v", err)
	}

	floatValue, ok := value.(float32)
	if !ok {
		t.Errorf("Expected float32 value, got %T", value)
	} else if floatValue != 42.5 {
		t.Errorf("Expected value 42.5, got %f", floatValue)
	}

	// Verify the correct tag was requested
	if mock.readCalls["R[5]"] != 1 {
		t.Errorf("Expected 1 read call for R[5], got %d", mock.readCalls["R[5]"])
	}

	// Set up mock response for DO register
	mock.readResponses["DO[10]"] = true

	// Test reading a DO register
	value, err = client.ReadRegister(RegisterTypeDO, 10)
	if err != nil {
		t.Errorf("Failed to read DO register: %v", err)
	}

	boolValue, ok := value.(bool)
	if !ok {
		t.Errorf("Expected bool value, got %T", value)
	} else if !boolValue {
		t.Errorf("Expected value true, got %v", boolValue)
	}
}

// TestWriteRegister tests the WriteRegister function
func TestWriteRegister(t *testing.T) {
	mock := newMockPLCClient()
	client := &FanucClient{PLCClient: mock}

	// Test writing to an R register
	err := client.WriteRegister(RegisterTypeR, 5, float32(42.5))
	if err != nil {
		t.Errorf("Failed to write R register: %v", err)
	}

	// Verify the correct value was written
	if mock.writeCalls["R[5]"] != float32(42.5) {
		t.Errorf("Expected value 42.5, got %v", mock.writeCalls["R[5]"])
	}

	// Test writing to a DO register
	err = client.WriteRegister(RegisterTypeDO, 10, true)
	if err != nil {
		t.Errorf("Failed to write DO register: %v", err)
	}

	// Verify the correct value was written
	if mock.writeCalls["DO[10]"] != true {
		t.Errorf("Expected value true, got %v", mock.writeCalls["DO[10]"])
	}
}

// TestReadPositionRegister tests the ReadPositionRegister function
func TestReadPositionRegister(t *testing.T) {
	mock := newMockPLCClient()
	client := &FanucClient{PLCClient: mock}

	// Set up mock responses for position register components
	mock.readResponses["PR[1].X"] = float32(100.1)
	mock.readResponses["PR[1].Y"] = float32(200.2)
	mock.readResponses["PR[1].Z"] = float32(300.3)
	mock.readResponses["PR[1].W"] = float32(0.0)
	mock.readResponses["PR[1].P"] = float32(90.0)
	mock.readResponses["PR[1].R"] = float32(180.0)
	mock.readResponses["PR[1].Config"] = "N U T, 0, 0, 0"
	mock.readResponses["PR[1].E1"] = float32(10.0)
	mock.readResponses["PR[1].E2"] = float32(20.0)

	// Test reading a position register
	position, err := client.ReadPositionRegister(1)
	if err != nil {
		t.Errorf("Failed to read position register: %v", err)
	}

	// Verify the position data
	if position.X != 100.1 {
		t.Errorf("Expected X = 100.1, got %f", position.X)
	}
	if position.Y != 200.2 {
		t.Errorf("Expected Y = 200.2, got %f", position.Y)
	}
	if position.Z != 300.3 {
		t.Errorf("Expected Z = 300.3, got %f", position.Z)
	}
	if position.W != 0.0 {
		t.Errorf("Expected W = 0.0, got %f", position.W)
	}
	if position.P != 90.0 {
		t.Errorf("Expected P = 90.0, got %f", position.P)
	}
	if position.R != 180.0 {
		t.Errorf("Expected R = 180.0, got %f", position.R)
	}
	if position.Config != "N U T, 0, 0, 0" {
		t.Errorf("Expected Config = 'N U T, 0, 0, 0', got %s", position.Config)
	}
	if len(position.Extensions) != 2 {
		t.Errorf("Expected 2 extension values, got %d", len(position.Extensions))
	}
	if position.Extensions[0] != 10.0 {
		t.Errorf("Expected E1 = 10.0, got %f", position.Extensions[0])
	}
	if position.Extensions[1] != 20.0 {
		t.Errorf("Expected E2 = 20.0, got %f", position.Extensions[1])
	}
}

// TestWritePositionRegister tests the WritePositionRegister function
func TestWritePositionRegister(t *testing.T) {
	mock := newMockPLCClient()
	client := &FanucClient{PLCClient: mock}

	// Create a position to write
	position := &Position{
		X:          100.1,
		Y:          200.2,
		Z:          300.3,
		W:          0.0,
		P:          90.0,
		R:          180.0,
		Config:     "N U T, 0, 0, 0",
		Extensions: []float32{10.0, 20.0},
	}

	// Test writing a position register
	err := client.WritePositionRegister(1, position)
	if err != nil {
		t.Errorf("Failed to write position register: %v", err)
	}

	// Verify each component was written correctly
	if mock.writeCalls["PR[1].X"] != 100.1 {
		t.Errorf("Expected X = 100.1, got %v", mock.writeCalls["PR[1].X"])
	}
	if mock.writeCalls["PR[1].Y"] != 200.2 {
		t.Errorf("Expected Y = 200.2, got %v", mock.writeCalls["PR[1].Y"])
	}
	if mock.writeCalls["PR[1].Z"] != 300.3 {
		t.Errorf("Expected Z = 300.3, got %v", mock.writeCalls["PR[1].Z"])
	}
	if mock.writeCalls["PR[1].W"] != 0.0 {
		t.Errorf("Expected W = 0.0, got %v", mock.writeCalls["PR[1].W"])
	}
	if mock.writeCalls["PR[1].P"] != 90.0 {
		t.Errorf("Expected P = 90.0, got %v", mock.writeCalls["PR[1].P"])
	}
	if mock.writeCalls["PR[1].R"] != 180.0 {
		t.Errorf("Expected R = 180.0, got %v", mock.writeCalls["PR[1].R"])
	}
	if mock.writeCalls["PR[1].Config"] != "N U T, 0, 0, 0" {
		t.Errorf("Expected Config = 'N U T, 0, 0, 0', got %v", mock.writeCalls["PR[1].Config"])
	}
	if mock.writeCalls["PR[1].E1"] != 10.0 {
		t.Errorf("Expected E1 = 10.0, got %v", mock.writeCalls["PR[1].E1"])
	}
	if mock.writeCalls["PR[1].E2"] != 20.0 {
		t.Errorf("Expected E2 = 20.0, got %v", mock.writeCalls["PR[1].E2"])
	}
}

// TestConvenienceMethods tests the convenience methods for register access
func TestConvenienceMethods(t *testing.T) {
	mock := newMockPLCClient()
	client := &FanucClient{PLCClient: mock}

	// Set up mock responses
	mock.readResponses["R[5]"] = float32(42.5)
	mock.readResponses["DI[10]"] = true

	// Test ReadRRegister
	value, err := client.ReadRRegister(5)
	if err != nil {
		t.Errorf("Failed to read R register: %v", err)
	}
	if value != 42.5 {
		t.Errorf("Expected value 42.5, got %f", value)
	}

	// Test ReadDIRegister
	boolValue, err := client.ReadDIRegister(10)
	if err != nil {
		t.Errorf("Failed to read DI register: %v", err)
	}
	if !boolValue {
		t.Errorf("Expected value true, got %v", boolValue)
	}

	// Test WriteRRegister
	err = client.WriteRRegister(5, 99.9)
	if err != nil {
		t.Errorf("Failed to write R register: %v", err)
	}
	if mock.writeCalls["R[5]"] != float32(99.9) {
		t.Errorf("Expected value 99.9, got %v", mock.writeCalls["R[5]"])
	}

	// Test WriteDORegister
	err = client.WriteDORegister(10, false)
	if err != nil {
		t.Errorf("Failed to write DO register: %v", err)
	}
	if mock.writeCalls["DO[10]"] != false {
		t.Errorf("Expected value false, got %v", mock.writeCalls["DO[10]"])
	}
}
