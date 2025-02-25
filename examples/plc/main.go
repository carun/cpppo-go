package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/carun/cpppo-go/pkg/cpppo"
)

// PLCClient provides a higher-level interface for PLC communication
type PLCClient struct {
	client *cpppo.Client
}

// NewPLCClient creates a new PLC client
func NewPLCClient(address string, timeout time.Duration) (*PLCClient, error) {
	client, err := cpppo.NewClient(address, timeout)
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
	request := cpppo.BuildCIPReadRequest(tagName, 1)

	// Send request
	response, err := p.client.SendRRData(0, 10, request)
	if err != nil {
		return nil, err
	}

	// Parse response
	return cpppo.ParseCIPReadResponse(response, dataType)
}

// WriteTag writes a value to a tag in the PLC
func (p *PLCClient) WriteTag(tagName string, dataType byte, value interface{}) error {
	var data []byte

	// Convert the value to the appropriate binary format based on data type
	switch dataType {
	case cpppo.CIPDataTypeBOOL:
		boolValue, ok := value.(bool)
		if !ok {
			return fmt.Errorf("value is not a bool")
		}
		if boolValue {
			data = []byte{1}
		} else {
			data = []byte{0}
		}

	case cpppo.CIPDataTypeSINT:
		intValue, ok := value.(int8)
		if !ok {
			return fmt.Errorf("value is not an int8")
		}
		data = []byte{byte(intValue)}

	case cpppo.CIPDataTypeINT:
		intValue, ok := value.(int16)
		if !ok {
			return fmt.Errorf("value is not an int16")
		}
		data = make([]byte, 2)
		binary.LittleEndian.PutUint16(data, uint16(intValue))

	case cpppo.CIPDataTypeDINT:
		intValue, ok := value.(int32)
		if !ok {
			return fmt.Errorf("value is not an int32")
		}
		data = make([]byte, 4)
		binary.LittleEndian.PutUint32(data, uint32(intValue))

	case cpppo.CIPDataTypeREAL:
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
	request := cpppo.BuildCIPWriteRequest(tagName, dataType, data)

	// Send request
	response, err := p.client.SendRRData(0, 10, request)
	if err != nil {
		return err
	}

	// Parse response to check for errors
	_, err = cpppo.ParseCIPResponse(response)
	return err
}

// Example usage
func main() {
	// Parse command-line arguments
	var ipAddress string
	var timeout time.Duration

	flag.StringVar(&ipAddress, "ip", "192.168.1.10", "IP address of the PLC/robot")
	flag.DurationVar(&timeout, "timeout", 5*time.Second, "Connection timeout")
	flag.Parse()

	fmt.Printf("Connecting to PLC at %s (timeout: %v)...\n", ipAddress, timeout)

	// Create a new PLC client
	plc, err := cpppo.NewPLCClient(ipAddress, timeout)
	if err != nil {
		log.Fatalf("Failed to create PLC client: %v", err)
	}
	defer plc.Close()

	// Read a DINT (double integer) tag
	fmt.Println("Reading Counter tag...")
	counterValue, err := plc.ReadTag("Program:MainProgram.Counter", cpppo.CIPDataTypeDINT)
	if err != nil {
		log.Fatalf("Failed to read tag: %v", err)
	}
	fmt.Printf("Counter value: %v\n", counterValue)

	// Read a BOOL tag
	fmt.Println("Reading Running tag...")
	runningValue, err := plc.ReadTag("Program:MainProgram.Running", cpppo.CIPDataTypeBOOL)
	if err != nil {
		log.Fatalf("Failed to read tag: %v", err)
	}
	fmt.Printf("Running value: %v\n", runningValue)

	// Write to a tag
	fmt.Println("Writing to SetPoint tag...")
	err = plc.WriteTag("Program:MainProgram.SetPoint", cpppo.CIPDataTypeREAL, float32(75.5))
	if err != nil {
		log.Fatalf("Failed to write tag: %v", err)
	}
	fmt.Println("Write successful")

	// Read back the value we just wrote
	fmt.Println("Reading back SetPoint tag...")
	setPointValue, err := plc.ReadTag("Program:MainProgram.SetPoint", cpppo.CIPDataTypeREAL)
	if err != nil {
		log.Fatalf("Failed to read tag: %v", err)
	}
	fmt.Printf("SetPoint value: %v\n", setPointValue)

	fmt.Println("Done")
}
