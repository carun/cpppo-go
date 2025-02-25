# CPPPO Go Library

A Go implementation of the CPPPO (Consortium of Python for Process and Protocol Operations) library for industrial protocol communications, with specialized support for FANUC robots.

## Overview

This library provides Go implementations of industrial protocols originally supported by the Python CPPPO library, with specialized features for FANUC robot communication. Currently, it supports:

- EtherNet/IP protocol communication
- CIP (Common Industrial Protocol) messaging
- Tag read/write operations
- Basic tag discovery and monitoring
- FANUC register access (R, PR, DI, DO, etc.)
- FANUC log reading and monitoring

## Installation

```
go get github.com/carun/cpppo-go
```

## Basic Usage

### Connecting to a PLC

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/carun/cpppo-go"
)

func main() {
	// Create a new client
	client, err := cpppo.NewClient("192.168.1.10", 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Register a session
	if err := client.RegisterSession(); err != nil {
		log.Fatalf("Failed to register session: %v", err)
	}

	fmt.Println("Successfully connected to PLC")
}
```

### Reading and Writing Tags

```go
// Read a tag
tagPath := cpppo.BuildCIPPath("Program:MainProgram.Counter")
readRequest := append([]byte{0x4C, 0x00}, tagPath...)
response, err := client.SendRRData(0, 10, readRequest)
if err != nil {
	log.Fatalf("Failed to read tag: %v", err)
}

// Parse the response
data, err := cpppo.ParseCIPResponse(response)
if err != nil {
	log.Fatalf("Failed to parse response: %v", err)
}

fmt.Printf("Tag value: %v\n", data)

// Write to a tag
writeRequest := cpppo.BuildCIPWriteRequest("Program:MainProgram.SetPoint", cpppo.CIPDataTypeREAL, []byte{0x00, 0x00, 0x96, 0x42}) // 75.0 as float32
response, err = client.SendRRData(0, 10, writeRequest)
if err != nil {
	log.Fatalf("Failed to write tag: %v", err)
}
```

### Using the Higher-Level API

The library includes a higher-level API for easier tag operations:

```go
// Create a PLC client
plc, err := NewPLCClient("192.168.1.10", 5*time.Second)
if err != nil {
	log.Fatalf("Failed to create PLC client: %v", err)
}
defer plc.Close()

// Read a tag
value, err := plc.ReadTag("Program:MainProgram.Counter", cpppo.CIPDataTypeDINT)
if err != nil {
	log.Fatalf("Failed to read tag: %v", err)
}
fmt.Printf("Counter value: %v\n", value)

// Write to a tag
err = plc.WriteTag("Program:MainProgram.SetPoint", cpppo.CIPDataTypeREAL, float32(75.5))
if err != nil {
	log.Fatalf("Failed to write tag: %v", err)
}
```

### FANUC Register Access

For FANUC robots, you can access registers directly:

```go
// Create a FANUC client
fanucClient, err := fanuc.NewFanucClient("192.168.1.10", 5*time.Second)
if err != nil {
	log.Fatalf("Failed to create FANUC client: %v", err)
}
defer fanucClient.Close()

// Read a numeric register (R)
value, err := fanucClient.ReadRRegister(1)
if err != nil {
	log.Fatalf("Failed to read R[1]: %v", err)
}
fmt.Printf("R[1] = %.2f\n", value)

// Read a position register (PR)
position, err := fanucClient.ReadPositionRegister(1)
if err != nil {
	log.Fatalf("Failed to read PR[1]: %v", err)
}
fmt.Printf("Position = X:%.2f Y:%.2f Z:%.2f W:%.2f P:%.2f R:%.2f\n",
    position.X, position.Y, position.Z, position.W, position.P, position.R)

// Write to a position register
newPosition := &fanuc.Position{
    X: 100.0, Y: 200.0, Z: 300.0,
    W: 0.0, P: 90.0, R: 0.0,
    Config: "N U T, 0, 0, 0",
}
err = fanucClient.WritePositionRegister(1, newPosition)
if err != nil {
	log.Fatalf("Failed to write to PR[1]: %v", err)
}
```

### FANUC Log Reading

You can also read logs from a FANUC controller:

```go
// Create a log reader
logReader := fanuc.NewLogReader("192.168.1.10", 5*time.Second)

// Get recent alarms
ctx := context.Background()
alarms, err := logReader.GetLatestAlarms(ctx, 10)
if err != nil {
	log.Fatalf("Failed to get alarms: %v", err)
}
for _, alarm := range alarms {
	fmt.Printf("[%s] [%s] %s\n", alarm.Timestamp, alarm.Code, alarm.Message)
}

// Monitor logs in real-time
request := fanuc.RemoteLogRequest{
    Types: []fanuc.LogType{fanuc.LogTypeAlarm, fanuc.LogTypeError},
    Since: time.Now().Add(-1 * time.Hour),
}

logs, err := logReader.StartRemoteLogMonitor(ctx, request)
if err != nil {
	log.Fatalf("Failed to start log monitoring: %v", err)
}

// Process logs as they arrive
for entry := range logs {
	fmt.Printf("[%s] [%s] %s\n", entry.Timestamp, entry.Type, entry.Message)
}
```

### Tag Monitoring

To continuously monitor tags:

```go
// Create a tag monitor
monitor := NewTagMonitor(plc, 1*time.Second)

// Add tags to monitor
monitor.AddTag("Program:MainProgram.Counter", cpppo.CIPDataTypeDINT)
monitor.AddTag("Program:MainProgram.Running", cpppo.CIPDataTypeBOOL)

// Register callbacks for value changes
monitor.OnChange("Program:MainProgram.Counter", func(tagName string, value interface{}) {
	fmt.Printf("Counter changed to %v\n", value)
})

// Start monitoring
monitor.Start()

// ... do other things ...

// Stop monitoring when done
monitor.Stop()
```

## Supported Features

### Core Protocol Features
- EtherNet/IP client communication
- Session management
- CIP messaging (read/write tags)
- Tag path construction
- Data type handling
- Value parsing
- Tag monitoring

### FANUC-Specific Features
- Access to FANUC registers (R, PR, DI, DO, AI, AO, etc.)
- Position register handling (X, Y, Z, W, P, R coordinates)
- Robot configuration and extended axis support
- Log monitoring (alarms, errors, events, etc.)
- Historical alarm retrieval
- Real-time log streaming

## Limitations

- Currently only supports EtherNet/IP and CIP protocols
- Limited tag discovery capabilities
- No support for array tags yet
- No server implementation
- Limited error handling for complex scenarios

## Comparison with Python CPPPO

This Go implementation focuses on the core features of the Python CPPPO library related to EtherNet/IP and CIP communications. It does not yet implement:

- MMS (Manufacturing Message Specification)
- IEC 60870-5 protocols
- Full server capabilities
- Advanced tag discovery
- All data types and structures

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This library is licensed under the MIT License - see the LICENSE.md file for details.
