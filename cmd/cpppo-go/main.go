package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/carun/cpppo-go/pkg/cpppo"
	"github.com/carun/cpppo-go/pkg/fanuc"
)

var (
	// Command line flags
	host     = flag.String("host", "127.0.0.1", "Host IP address")
	port     = flag.Int("port", 44818, "Port number (default: 44818 for EtherNet/IP)")
	timeout  = flag.Duration("timeout", 5*time.Second, "Connection timeout")
	mode     = flag.String("mode", "info", "Operation mode (info, read, write, logs)")
	tag      = flag.String("tag", "", "Tag name to read/write")
	dataType = flag.String("type", "DINT", "Data type (BOOL, SINT, INT, DINT, REAL)")
	value    = flag.String("value", "", "Value to write (for write mode)")
	register = flag.Int("register", 0, "Register number (for FANUC mode)")
	regType  = flag.String("regtype", "R", "Register type (R, PR, DI, DO, etc.)")
	logType  = flag.String("logtype", "ALARM", "Log type to monitor (ALARM, ERROR, EVENT, etc.)")
	fanucOpt = flag.Bool("fanuc", false, "Use FANUC-specific features")
)

func main() {
	// Parse command line flags
	flag.Parse()

	// Construct the address
	address := fmt.Sprintf("%s:%d", *host, *port)

	// Choose between FANUC and standard modes
	if *fanucOpt {
		runFanucMode(address)
	} else {
		runStandardMode(address)
	}
}

func runStandardMode(address string) {
	// Create a new client
	fmt.Printf("Connecting to %s...\n", address)
	client, err := cpppo.NewClient(address, *timeout)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Register a session
	if err := client.RegisterSession(); err != nil {
		log.Fatalf("Failed to register session: %v", err)
	}
	fmt.Println("Session registered successfully")

	// Execute the requested operation
	switch *mode {
	case "info":
		// List Identity
		fmt.Println("Sending List Identity request...")
		data, err := client.ListIdentity()
		if err != nil {
			log.Fatalf("Failed to list identity: %v", err)
		}
		fmt.Printf("Received identity data of length: %d bytes\n", len(data))
		// TODO: Add parsing of identity data

	case "read":
		if *tag == "" {
			log.Fatalf("Tag name is required for read mode")
		}

		// Create a PLC client for higher-level operations
		plcClient := &PLCClient{client: client}

		// Convert string data type to byte
		dataTypeByte := getDataTypeByte(*dataType)

		// Read the tag
		fmt.Printf("Reading tag %s of type %s...\n", *tag, *dataType)
		value, err := plcClient.ReadTag(*tag, dataTypeByte)
		if err != nil {
			log.Fatalf("Failed to read tag: %v", err)
		}
		fmt.Printf("Value: %v\n", value)

	case "write":
		if *tag == "" || *value == "" {
			log.Fatalf("Tag name and value are required for write mode")
		}

		// Create a PLC client for higher-level operations
		plcClient := &PLCClient{client: client}

		// Convert string data type to byte
		dataTypeByte := getDataTypeByte(*dataType)

		// Convert string value to appropriate type
		typedValue, err := convertValue(*value, *dataType)
		if err != nil {
			log.Fatalf("Failed to convert value: %v", err)
		}

		// Write to the tag
		fmt.Printf("Writing value %v to tag %s of type %s...\n", typedValue, *tag, *dataType)
		err = plcClient.WriteTag(*tag, dataTypeByte, typedValue)
		if err != nil {
			log.Fatalf("Failed to write tag: %v", err)
		}
		fmt.Println("Write successful")

	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func runFanucMode(address string) {
	// Create a FANUC client
	fmt.Printf("Connecting to FANUC controller at %s...\n", address)
	client, err := fanuc.NewFanucClient(address, *timeout)
	if err != nil {
		log.Fatalf("Failed to connect to FANUC controller: %v", err)
	}
	defer client.Close()
	fmt.Println("Connected to FANUC controller")

	// Execute the requested operation
	switch *mode {
	case "info":
		fmt.Println("FANUC controller connected successfully")
		// You might want to add more info here in the future

	case "read":
		// Check if we're reading a register
		if *register > 0 {
			// Convert string register type to RegisterType
			regTypeEnum := getRegisterType(*regType)

			// Read the register
			fmt.Printf("Reading %s[%d]...\n", *regType, *register)
			value, err := client.ReadRegister(regTypeEnum, *register)
			if err != nil {
				log.Fatalf("Failed to read register: %v", err)
			}
			fmt.Printf("Value: %v\n", value)
		} else if *tag != "" {
			// Or read a tag if specified
			dataTypeByte := getDataTypeByte(*dataType)
			fmt.Printf("Reading tag %s of type %s...\n", *tag, *dataType)
			value, err := client.PLCClient.ReadTag(*tag, dataTypeByte)
			if err != nil {
				log.Fatalf("Failed to read tag: %v", err)
			}
			fmt.Printf("Value: %v\n", value)
		} else {
			log.Fatalf("Either register or tag must be specified for read mode")
		}

	case "write":
		// Check if we're writing to a register
		if *register > 0 && *value != "" {
			// Convert string register type to RegisterType
			regTypeEnum := getRegisterType(*regType)

			// Convert string value to appropriate type
			var typedValue interface{}
			var err error

			// Handle position registers specially
			if regTypeEnum == fanuc.RegisterTypePR {
				log.Fatalf("Writing to position registers from CLI is not supported yet")
			} else {
				// For other register types, convert the value
				switch regTypeEnum {
				case fanuc.RegisterTypeR:
					typedValue, err = convertValue(*value, "REAL")
				case fanuc.RegisterTypeDI, fanuc.RegisterTypeDO:
					typedValue, err = convertValue(*value, "BOOL")
				default:
					typedValue, err = convertValue(*value, *dataType)
				}

				if err != nil {
					log.Fatalf("Failed to convert value: %v", err)
				}

				// Write to the register
				fmt.Printf("Writing value %v to %s[%d]...\n", typedValue, *regType, *register)
				err = client.WriteRegister(regTypeEnum, *register, typedValue)
				if err != nil {
					log.Fatalf("Failed to write to register: %v", err)
				}
				fmt.Println("Write successful")
			}
		} else if *tag != "" && *value != "" {
			// Or write to a tag if specified
			dataTypeByte := getDataTypeByte(*dataType)
			typedValue, err := convertValue(*value, *dataType)
			if err != nil {
				log.Fatalf("Failed to convert value: %v", err)
			}
			fmt.Printf("Writing value %v to tag %s of type %s...\n", typedValue, *tag, *dataType)
			err = client.PLCClient.WriteTag(*tag, dataTypeByte, typedValue)
			if err != nil {
				log.Fatalf("Failed to write tag: %v", err)
			}
			fmt.Println("Write successful")
		} else {
			log.Fatalf("Either register or tag, and a value must be specified for write mode")
		}

	case "logs":
		// Create a log reader
		logReader := fanuc.NewLogReader(address, *timeout)

		// Get the log type
		logTypeEnum := fanuc.LogType(*logType)

		// Get the latest alarms
		fmt.Println("Getting the latest alarms...")
		ctx := context.Background()
		alarms, err := logReader.GetLatestAlarms(ctx, 10)
		if err != nil {
			log.Fatalf("Failed to get alarms: %v", err)
		}

		fmt.Println("Latest alarms:")
		for i, alarm := range alarms {
			fmt.Printf("%d. [%s] [%s] %s\n", i+1, alarm.Timestamp.Format("2006-01-02 15:04:05"), alarm.Code, alarm.Message)
		}

		// Monitor logs for 30 seconds
		fmt.Printf("Monitoring logs of type %s for 30 seconds...\n", logTypeEnum)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		request := fanuc.RemoteLogRequest{
			Types: []fanuc.LogType{logTypeEnum},
			Since: time.Now().Add(-1 * time.Hour),
		}

		logs, err := logReader.StartRemoteLogMonitor(ctx, request)
		if err != nil {
			log.Fatalf("Failed to start log monitoring: %v", err)
		}

		// Process logs as they arrive
		for entry := range logs {
			fmt.Printf("[%s] [%s] %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"), entry.Type, entry.Message)
		}

		fmt.Println("Log monitoring complete")

	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

// Helper functions

// PLCClient is a simplified version for this example
type PLCClient struct {
	client *cpppo.Client
}

func (p *PLCClient) ReadTag(tagName string, dataType byte) (interface{}, error) {
	// This is a simplified implementation
	// In a real app, this would use the proper implementation from the library
	// Create CIP read request
	request := []byte{0x4C, 0x00} // Service code for Read Tag
	// Add path and other data...
	// Just a placeholder for this example
	response, err := p.client.SendRRData(0, 10, request)
	if err != nil {
		return nil, err
	}
	if len(response) > 0 {
		fmt.Println("Received response of length:", len(response))
	}
	// Parse response and return value
	// This is just a placeholder
	return "value", nil
}

func (p *PLCClient) WriteTag(tagName string, dataType byte, value interface{}) error {
	// This is a simplified implementation
	// In a real app, this would use the proper implementation from the library
	// Create CIP write request
	request := []byte{0x4D, 0x00} // Service code for Write Tag
	// Add path, data type, value, etc.
	// Just a placeholder for this example
	_, err := p.client.SendRRData(0, 10, request)
	return err
}

func getDataTypeByte(dataType string) byte {
	switch dataType {
	case "BOOL":
		return cpppo.CIPDataTypeBOOL
	case "SINT":
		return cpppo.CIPDataTypeSINT
	case "INT":
		return cpppo.CIPDataTypeINT
	case "DINT":
		return cpppo.CIPDataTypeDINT
	case "REAL":
		return cpppo.CIPDataTypeREAL
	case "DWORD":
		return cpppo.CIPDataTypeDWORD
	case "STRING":
		return cpppo.CIPDataTypeSTRING
	default:
		log.Fatalf("Unsupported data type: %s", dataType)
		return 0
	}
}

func getRegisterType(regType string) fanuc.RegisterType {
	switch regType {
	case "R":
		return fanuc.RegisterTypeR
	case "PR":
		return fanuc.RegisterTypePR
	case "DI":
		return fanuc.RegisterTypeDI
	case "DO":
		return fanuc.RegisterTypeDO
	case "AI":
		return fanuc.RegisterTypeAI
	case "AO":
		return fanuc.RegisterTypeAO
	case "GI":
		return fanuc.RegisterTypeGI
	case "GO":
		return fanuc.RegisterTypeGO
	case "UR":
		return fanuc.RegisterTypeUR
	case "SR":
		return fanuc.RegisterTypeSR
	case "VR":
		return fanuc.RegisterTypeVR
	default:
		log.Fatalf("Unsupported register type: %s", regType)
		return fanuc.RegisterTypeR
	}
}

func convertValue(valueStr string, dataType string) (interface{}, error) {
	switch dataType {
	case "BOOL":
		return valueStr == "true" || valueStr == "1", nil
	case "SINT":
		// Parse as int8
		i, err := strconv.ParseInt(valueStr, 10, 8)
		return int8(i), err
	case "INT":
		// Parse as int16
		i, err := strconv.ParseInt(valueStr, 10, 16)
		return int16(i), err
	case "DINT":
		// Parse as int32
		i, err := strconv.ParseInt(valueStr, 10, 32)
		return int32(i), err
	case "REAL":
		// Parse as float32
		f, err := strconv.ParseFloat(valueStr, 32)
		return float32(f), err
	case "DWORD":
		// Parse as uint32
		i, err := strconv.ParseUint(valueStr, 10, 32)
		return uint32(i), err
	case "STRING":
		return valueStr, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
}
