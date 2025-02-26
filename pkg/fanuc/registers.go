package fanuc

import (
	"errors"
	"fmt"
	"time"

	"github.com/carun/cpppo-go/pkg/cpppo"
)

// RegisterType represents Fanuc register types
type RegisterType int

const (
	// Register types in Fanuc controllers
	RegisterTypeR  RegisterType = iota // R registers (for numerical data)
	RegisterTypePR                     // PR registers (for position registers)
	RegisterTypeDI                     // DI registers (for digital inputs)
	RegisterTypeDO                     // DO registers (for digital outputs)
	RegisterTypeAI                     // AI registers (for analog inputs)
	RegisterTypeAO                     // AO registers (for analog outputs)
	RegisterTypeGI                     // GI registers (for group inputs)
	RegisterTypeGO                     // GO registers (for group outputs)
	RegisterTypeUR                     // UR registers (for user frame registers)
	RegisterTypeSR                     // SR registers (for string registers)
	RegisterTypeVR                     // VR registers (for vision registers)
)

// PLCClientInterface defines the interface for PLC client implementations
type PLCClientInterface interface {
	ReadTag(tagName string, dataType byte) (interface{}, error)
	WriteTag(tagName string, dataType byte, value interface{}) error
	Close() error
}

// FanucClient extends the PLC client with Fanuc-specific functionality
type FanucClient struct {
	PLCClient PLCClientInterface
}

// NewFanucClient creates a new Fanuc client
func NewFanucClient(address string, timeout time.Duration) (*FanucClient, error) {
	plcClient, err := cpppo.NewPLCClient(address, timeout)
	if err != nil {
		return nil, err
	}

	return &FanucClient{
		PLCClient: plcClient,
	}, nil
}

// Close closes the Fanuc client
func (f *FanucClient) Close() error {
	return f.PLCClient.Close()
}

// buildRegisterTag creates the CIP-compatible tag name for a Fanuc register
func buildRegisterTag(regType RegisterType, index int) string {
	switch regType {
	case RegisterTypeR:
		return fmt.Sprintf("R[%d]", index)
	case RegisterTypePR:
		return fmt.Sprintf("PR[%d]", index)
	case RegisterTypeDI:
		return fmt.Sprintf("DI[%d]", index)
	case RegisterTypeDO:
		return fmt.Sprintf("DO[%d]", index)
	case RegisterTypeAI:
		return fmt.Sprintf("AI[%d]", index)
	case RegisterTypeAO:
		return fmt.Sprintf("AO[%d]", index)
	case RegisterTypeGI:
		return fmt.Sprintf("GI[%d]", index)
	case RegisterTypeGO:
		return fmt.Sprintf("GO[%d]", index)
	case RegisterTypeUR:
		return fmt.Sprintf("UR[%d]", index)
	case RegisterTypeSR:
		return fmt.Sprintf("SR[%d]", index)
	case RegisterTypeVR:
		return fmt.Sprintf("VR[%d]", index)
	default:
		return fmt.Sprintf("R[%d]", index)
	}
}

// getRegisterDataType returns the CIP data type for a register type
func getRegisterDataType(regType RegisterType) byte {
	switch regType {
	case RegisterTypeR:
		return cpppo.CIPDataTypeREAL
	case RegisterTypePR:
		// Position registers are complex structures
		// For simplicity, we'll use string to get raw data
		return cpppo.CIPDataTypeSTRING
	case RegisterTypeDI, RegisterTypeDO, RegisterTypeGI, RegisterTypeGO:
		return cpppo.CIPDataTypeBOOL
	case RegisterTypeAI, RegisterTypeAO:
		return cpppo.CIPDataTypeREAL
	case RegisterTypeUR:
		// User frames are complex structures
		return cpppo.CIPDataTypeSTRING
	case RegisterTypeSR:
		return cpppo.CIPDataTypeSTRING
	case RegisterTypeVR:
		// Vision registers can be complex
		return cpppo.CIPDataTypeSTRING
	default:
		return cpppo.CIPDataTypeREAL
	}
}

// ReadRegister reads a value from a Fanuc register
func (f *FanucClient) ReadRegister(regType RegisterType, index int) (interface{}, error) {
	// Build the tag name for the register
	tagName := buildRegisterTag(regType, index)

	// Get the data type for this register type
	dataType := getRegisterDataType(regType)

	// Special handling for position registers (PR) which have components
	if regType == RegisterTypePR {
		return f.ReadPositionRegister(index)
	}

	// Read the register using the PLC client
	return f.PLCClient.ReadTag(tagName, dataType)
}

// Position represents a position in Cartesian space
type Position struct {
	X, Y, Z    float32   // Cartesian coordinates
	W, P, R    float32   // Wrist orientation (W/P/R format)
	Config     string    // Robot configuration
	Extensions []float32 // Additional axes
}

// ReadPositionRegister reads a position register (PR) and returns structured data
func (f *FanucClient) ReadPositionRegister(index int) (*Position, error) {
	// Position registers have multiple components
	// We need to read each component separately

	// Read X component
	xValue, err := f.PLCClient.ReadTag(fmt.Sprintf("PR[%d].X", index), cpppo.CIPDataTypeREAL)
	if err != nil {
		return nil, fmt.Errorf("failed to read PR X component: %w", err)
	}

	// Read Y component
	yValue, err := f.PLCClient.ReadTag(fmt.Sprintf("PR[%d].Y", index), cpppo.CIPDataTypeREAL)
	if err != nil {
		return nil, fmt.Errorf("failed to read PR Y component: %w", err)
	}

	// Read Z component
	zValue, err := f.PLCClient.ReadTag(fmt.Sprintf("PR[%d].Z", index), cpppo.CIPDataTypeREAL)
	if err != nil {
		return nil, fmt.Errorf("failed to read PR Z component: %w", err)
	}

	// Read W component
	wValue, err := f.PLCClient.ReadTag(fmt.Sprintf("PR[%d].W", index), cpppo.CIPDataTypeREAL)
	if err != nil {
		return nil, fmt.Errorf("failed to read PR W component: %w", err)
	}

	// Read P component
	pValue, err := f.PLCClient.ReadTag(fmt.Sprintf("PR[%d].P", index), cpppo.CIPDataTypeREAL)
	if err != nil {
		return nil, fmt.Errorf("failed to read PR P component: %w", err)
	}

	// Read R component
	rValue, err := f.PLCClient.ReadTag(fmt.Sprintf("PR[%d].R", index), cpppo.CIPDataTypeREAL)
	if err != nil {
		return nil, fmt.Errorf("failed to read PR R component: %w", err)
	}

	// Read config string
	configValue, err := f.PLCClient.ReadTag(fmt.Sprintf("PR[%d].Config", index), cpppo.CIPDataTypeSTRING)
	if err != nil {
		return nil, fmt.Errorf("failed to read PR Config component: %w", err)
	}

	// Create position struct
	position := &Position{
		X:      xValue.(float32),
		Y:      yValue.(float32),
		Z:      zValue.(float32),
		W:      wValue.(float32),
		P:      pValue.(float32),
		R:      rValue.(float32),
		Config: configValue.(string),
	}

	// Try to read extension axes if they exist
	// This is controller-dependent, so we'll try E1-E3 and ignore errors
	extensions := []float32{}

	for i := 1; i <= 3; i++ {
		eValue, err := f.PLCClient.ReadTag(fmt.Sprintf("PR[%d].E%d", index, i), cpppo.CIPDataTypeREAL)
		if err == nil {
			extensions = append(extensions, eValue.(float32))
		}
	}

	position.Extensions = extensions

	return position, nil
}

// WriteRegister writes a value to a Fanuc register
func (f *FanucClient) WriteRegister(regType RegisterType, index int, value interface{}) error {
	// Build the tag name for the register
	tagName := buildRegisterTag(regType, index)

	// Get the data type for this register type
	dataType := getRegisterDataType(regType)

	// Special handling for position registers (PR)
	if regType == RegisterTypePR {
		pos, ok := value.(*Position)
		if !ok {
			return errors.New("value must be a Position for PR registers")
		}
		return f.WritePositionRegister(index, pos)
	}

	// Write the register using the PLC client
	return f.PLCClient.WriteTag(tagName, dataType, value)
}

// WritePositionRegister writes a Position to a position register (PR)
func (f *FanucClient) WritePositionRegister(index int, position *Position) error {
	// Write each component separately

	// Write X component
	err := f.PLCClient.WriteTag(fmt.Sprintf("PR[%d].X", index), cpppo.CIPDataTypeREAL, position.X)
	if err != nil {
		return fmt.Errorf("failed to write PR X component: %w", err)
	}

	// Write Y component
	err = f.PLCClient.WriteTag(fmt.Sprintf("PR[%d].Y", index), cpppo.CIPDataTypeREAL, position.Y)
	if err != nil {
		return fmt.Errorf("failed to write PR Y component: %w", err)
	}

	// Write Z component
	err = f.PLCClient.WriteTag(fmt.Sprintf("PR[%d].Z", index), cpppo.CIPDataTypeREAL, position.Z)
	if err != nil {
		return fmt.Errorf("failed to write PR Z component: %w", err)
	}

	// Write W component
	err = f.PLCClient.WriteTag(fmt.Sprintf("PR[%d].W", index), cpppo.CIPDataTypeREAL, position.W)
	if err != nil {
		return fmt.Errorf("failed to write PR W component: %w", err)
	}

	// Write P component
	err = f.PLCClient.WriteTag(fmt.Sprintf("PR[%d].P", index), cpppo.CIPDataTypeREAL, position.P)
	if err != nil {
		return fmt.Errorf("failed to write PR P component: %w", err)
	}

	// Write R component
	err = f.PLCClient.WriteTag(fmt.Sprintf("PR[%d].R", index), cpppo.CIPDataTypeREAL, position.R)
	if err != nil {
		return fmt.Errorf("failed to write PR R component: %w", err)
	}

	// Write Config
	err = f.PLCClient.WriteTag(fmt.Sprintf("PR[%d].Config", index), cpppo.CIPDataTypeSTRING, position.Config)
	if err != nil {
		return fmt.Errorf("failed to write PR Config component: %w", err)
	}

	// Write extension axes if they exist
	for i, ext := range position.Extensions {
		if i >= 3 {
			break // Only support up to 3 extension axes
		}

		err = f.PLCClient.WriteTag(fmt.Sprintf("PR[%d].E%d", index, i+1), cpppo.CIPDataTypeREAL, ext)
		if err != nil {
			return fmt.Errorf("failed to write PR E%d component: %w", i+1, err)
		}
	}

	return nil
}

// ReadRRegister is a convenience method to read an R register
func (f *FanucClient) ReadRRegister(index int) (float32, error) {
	value, err := f.ReadRegister(RegisterTypeR, index)
	if err != nil {
		return 0, err
	}
	floatVal, ok := value.(float32)
	if !ok {
		return 0, errors.New("failed to convert value to float32")
	}
	return floatVal, nil
}

// WriteRRegister is a convenience method to write an R register
func (f *FanucClient) WriteRRegister(index int, value float32) error {
	return f.WriteRegister(RegisterTypeR, index, value)
}

// ReadDIRegister is a convenience method to read a DI register
func (f *FanucClient) ReadDIRegister(index int) (bool, error) {
	value, err := f.ReadRegister(RegisterTypeDI, index)
	if err != nil {
		return false, err
	}
	boolVal, ok := value.(bool)
	if !ok {
		return false, errors.New("failed to convert value to bool")
	}
	return boolVal, nil
}

// WriteDORegister is a convenience method to write a DO register
func (f *FanucClient) WriteDORegister(index int, value bool) error {
	return f.WriteRegister(RegisterTypeDO, index, value)
}
