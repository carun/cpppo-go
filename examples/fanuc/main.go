package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/carun/cpppo-go/pkg/fanuc"
)

func main() {
	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up a signal handler to gracefully shut down
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// Parse command-line arguments
	var fanucIP string
	var timeout time.Duration

	flag.StringVar(&fanucIP, "ip", "192.168.1.10", "IP address of the PLC/robot")
	flag.DurationVar(&timeout, "timeout", 5*time.Second, "Connection timeout")
	flag.Parse()

	fmt.Printf("Connecting to PLC at %s (timeout: %v)...\n", fanucIP, timeout)

	// Initialize Fanuc client
	fmt.Printf("Connecting to Fanuc controller at %s...\n", fanucIP)
	client, err := fanuc.NewFanucClient(fanucIP, timeout)
	if err != nil {
		log.Fatalf("Failed to connect to Fanuc controller: %v", err)
	}
	defer client.Close()
	fmt.Println("Successfully connected to Fanuc controller")

	// Initialize log reader
	logReader := fanuc.NewLogReader(fanucIP, timeout)

	// Wait group to coordinate goroutines
	var wg sync.WaitGroup

	// Start register monitoring
	wg.Add(1)
	go monitorRegisters(ctx, &wg, client)

	// Start log monitoring
	wg.Add(1)
	go monitorLogs(ctx, &wg, logReader)

	// Wait for all goroutines to complete
	wg.Wait()
	fmt.Println("Application shutdown complete")
}

// monitorRegisters continuously monitors and logs register values
func monitorRegisters(ctx context.Context, wg *sync.WaitGroup, client *fanuc.FanucClient) {
	defer wg.Done()

	fmt.Println("Starting register monitoring...")

	// These are common registers you might want to monitor
	// Adjust based on your specific robot program
	registerIndices := []int{1, 2, 3, 4, 5}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Register monitoring stopped")
			return
		case <-ticker.C:
			// Read numeric registers (R)
			for _, index := range registerIndices {
				value, err := client.ReadRRegister(index)
				if err != nil {
					fmt.Printf("Error reading R[%d]: %v\n", index, err)
					continue
				}
				fmt.Printf("R[%d] = %.2f\n", index, value)
			}

			// Read a position register (PR)
			prIndex := 1 // First position register
			position, err := client.ReadPositionRegister(prIndex)
			if err != nil {
				fmt.Printf("Error reading PR[%d]: %v\n", prIndex, err)
			} else {
				fmt.Printf("PR[%d] = X:%.2f Y:%.2f Z:%.2f W:%.2f P:%.2f R:%.2f Config:%s\n",
					prIndex, position.X, position.Y, position.Z, position.W, position.P, position.R, position.Config)
			}

			// Read digital inputs/outputs
			for i := 1; i <= 5; i++ {
				di, err := client.ReadDIRegister(i)
				if err != nil {
					fmt.Printf("Error reading DI[%d]: %v\n", i, err)
				} else {
					fmt.Printf("DI[%d] = %v\n", i, di)
				}
			}

			fmt.Println("-------------------------------------------")
		}
	}
}

// monitorLogs reads and displays log entries from the Fanuc controller
func monitorLogs(ctx context.Context, wg *sync.WaitGroup, logReader *fanuc.LogReader) {
	defer wg.Done()

	fmt.Println("Starting log monitoring...")

	// First, get recent alarms
	alarms, err := logReader.GetLatestAlarms(ctx, 10)
	if err != nil {
		fmt.Printf("Error getting latest alarms: %v\n", err)
	} else {
		fmt.Println("Latest alarms:")
		for i, alarm := range alarms {
			fmt.Printf("%d. [%s] [%s] %s\n", i+1, alarm.Timestamp.Format("2006-01-02 15:04:05"), alarm.Code, alarm.Message)
		}
		fmt.Println("-------------------------------------------")
	}

	// Set up remote log monitoring for specific log types
	request := fanuc.RemoteLogRequest{
		Types: []fanuc.LogType{
			fanuc.LogTypeAlarm,
			fanuc.LogTypeError,
			fanuc.LogTypeEvent,
		},
		Since: time.Now().Add(-1 * time.Hour), // Only get logs from the last hour
	}

	logs, err := logReader.StartRemoteLogMonitor(ctx, request)
	if err != nil {
		fmt.Printf("Error starting log monitoring: %v\n", err)
		return
	}

	// Process log entries
	for {
		select {
		case <-ctx.Done():
			logReader.StopRemoteLogMonitor()
			fmt.Println("Log monitoring stopped")
			return
		case entry, ok := <-logs:
			if !ok {
				fmt.Println("Log channel closed")
				return
			}

			// Format log entry based on its type
			switch entry.Type {
			case fanuc.LogTypeAlarm:
				fmt.Printf("ALARM: [%s] [%s] %s\n", entry.Timestamp.Format("15:04:05"), entry.Code, entry.Message)
			case fanuc.LogTypeError:
				fmt.Printf("ERROR: [%s] %s\n", entry.Timestamp.Format("15:04:05"), entry.Message)
			default:
				fmt.Printf("LOG: [%s] [%s] %s\n", entry.Timestamp.Format("15:04:05"), entry.Type, entry.Message)
			}
		}
	}
}

// writeToPositionRegister demonstrates writing to a position register
func writeToPositionRegister(client *fanuc.FanucClient) {
	// Create a new position
	newPosition := &fanuc.Position{
		X:      250.0,
		Y:      150.0,
		Z:      50.0,
		W:      180.0,
		P:      0.0,
		R:      0.0,
		Config: "N U T, 0, 0, 0", // Sample configuration string
	}

	// Write to position register 1
	err := client.WritePositionRegister(1, newPosition)
	if err != nil {
		fmt.Printf("Error writing to PR[1]: %v\n", err)
		return
	}

	fmt.Println("Successfully wrote to PR[1]")

	// Read back and verify
	position, err := client.ReadPositionRegister(1)
	if err != nil {
		fmt.Printf("Error reading PR[1]: %v\n", err)
		return
	}

	fmt.Printf("PR[1] now = X:%.2f Y:%.2f Z:%.2f W:%.2f P:%.2f R:%.2f Config:%s\n",
		position.X, position.Y, position.Z, position.W, position.P, position.R, position.Config)
}

// demonstrateRegisterOperations shows various register operations
func demonstrateRegisterOperations(client *fanuc.FanucClient) {
	// Read various register types
	fmt.Println("Reading registers...")

	// Read R registers (numeric)
	for i := 1; i <= 5; i++ {
		value, err := client.ReadRRegister(i)
		if err != nil {
			fmt.Printf("Error reading R[%d]: %v\n", i, err)
		} else {
			fmt.Printf("R[%d] = %.2f\n", i, value)
		}
	}

	// Write to R register
	fmt.Println("Writing to R[1]...")
	err := client.WriteRRegister(1, 99.5)
	if err != nil {
		fmt.Printf("Error writing to R[1]: %v\n", err)
	} else {
		fmt.Println("Successfully wrote 99.5 to R[1]")
	}

	// Read back to verify
	value, err := client.ReadRRegister(1)
	if err != nil {
		fmt.Printf("Error reading R[1]: %v\n", err)
	} else {
		fmt.Printf("R[1] now = %.2f\n", value)
	}

	// Write to digital output
	fmt.Println("Setting DO[1] to ON...")
	err = client.WriteDORegister(1, true)
	if err != nil {
		fmt.Printf("Error writing to DO[1]: %v\n", err)
	} else {
		fmt.Println("Successfully set DO[1] to ON")
	}
}
