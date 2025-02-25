package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/carun/cpppo-go/pkg/cpppo"
)

// TagMonitor monitors PLC tags at specified intervals
type TagMonitor struct {
	plc       *cpppo.PLCClient
	interval  time.Duration
	tags      map[string]byte // map of tag names to their data types
	values    map[string]interface{}
	callbacks map[string][]func(string, interface{})
	stopChan  chan struct{}
	mu        sync.RWMutex
	wg        sync.WaitGroup
}

// NewTagMonitor creates a new tag monitor
func NewTagMonitor(plc *cpppo.PLCClient, interval time.Duration) *TagMonitor {
	return &TagMonitor{
		plc:       plc,
		interval:  interval,
		tags:      make(map[string]byte),
		values:    make(map[string]interface{}),
		callbacks: make(map[string][]func(string, interface{})),
		stopChan:  make(chan struct{}),
	}
}

// AddTag adds a tag to monitor
func (m *TagMonitor) AddTag(tagName string, dataType byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tags[tagName] = dataType
}

// GetValue gets the last known value of a tag
func (m *TagMonitor) GetValue(tagName string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.values[tagName]
	return value, ok
}

// OnChange registers a callback to be called when a tag value changes
func (m *TagMonitor) OnChange(tagName string, callback func(string, interface{})) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks[tagName] = append(m.callbacks[tagName], callback)
}

// Start starts monitoring tags
func (m *TagMonitor) Start() {
	m.wg.Add(1)
	go m.monitorLoop()
}

// Stop stops monitoring tags
func (m *TagMonitor) Stop() {
	close(m.stopChan)
	m.wg.Wait()
}

// monitorLoop is the main monitoring loop
func (m *TagMonitor) monitorLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.pollTags()
		}
	}
}

// pollTags polls all tags and updates values
func (m *TagMonitor) pollTags() {
	m.mu.RLock()
	tagsCopy := make(map[string]byte, len(m.tags))
	for tag, dataType := range m.tags {
		tagsCopy[tag] = dataType
	}
	m.mu.RUnlock()

	for tagName, dataType := range tagsCopy {
		value, err := m.plc.ReadTag(tagName, dataType)
		if err != nil {
			log.Printf("Error reading tag %s: %v", tagName, err)
			continue
		}

		m.mu.Lock()
		oldValue, exists := m.values[tagName]
		valueChanged := !exists || !valueEquals(oldValue, value)

		if valueChanged {
			m.values[tagName] = value
			// Make a copy of callbacks to avoid holding the lock during callback execution
			callbacks := make([]func(string, interface{}), len(m.callbacks[tagName]))
			copy(callbacks, m.callbacks[tagName])
			m.mu.Unlock()

			// Execute callbacks outside the lock
			for _, callback := range callbacks {
				callback(tagName, value)
			}
		} else {
			m.mu.Unlock()
		}
	}
}

// valueEquals compares two values for equality
func valueEquals(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == b
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// TagDiscovery discovers tags in a PLC
type TagDiscovery struct {
	plc *cpppo.PLCClient
}

// NewTagDiscovery creates a new tag discovery
func NewTagDiscovery(plc *cpppo.PLCClient) *TagDiscovery {
	return &TagDiscovery{
		plc: plc,
	}
}

// DiscoverTags attempts to discover tags in the given program
// This is a simplified implementation and won't work with all PLCs
func (d *TagDiscovery) DiscoverTags(programName string) ([]string, error) {
	// This is a placeholder for tag discovery
	// Real implementation would depend on the specific PLC and protocol
	// Some PLCs support reading a tag list, others require browsing objects

	// For demonstration purposes, we'll simulate discovering some common tags
	tags := []string{
		programName + ".Counter",
		programName + ".SetPoint",
		programName + ".Running",
		programName + ".Status",
		programName + ".Temperature",
	}

	return tags, nil
}

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

	fmt.Println("Successfully connected to PLC")

	// Discover tags
	discovery := NewTagDiscovery(plc)
	tags, err := discovery.DiscoverTags("Program:MainProgram")
	if err != nil {
		log.Fatalf("Failed to discover tags: %v", err)
	}

	fmt.Println("Discovered tags:")
	for _, tag := range tags {
		fmt.Println(" -", tag)
	}

	// Create a tag monitor
	monitor := NewTagMonitor(plc, 1*time.Second)

	// Add discovered tags with assumed data types
	// In a real implementation, you'd determine the data type for each tag
	for _, tag := range tags {
		var dataType byte

		// Guess the data type based on tag name (just for demonstration)
		switch {
		case tag == "Program:MainProgram.Counter":
			dataType = cpppo.CIPDataTypeDINT
		case tag == "Program:MainProgram.SetPoint":
			dataType = cpppo.CIPDataTypeREAL
		case tag == "Program:MainProgram.Running":
			dataType = cpppo.CIPDataTypeBOOL
		case tag == "Program:MainProgram.Status":
			dataType = cpppo.CIPDataTypeINT
		case tag == "Program:MainProgram.Temperature":
			dataType = cpppo.CIPDataTypeREAL
		default:
			dataType = cpppo.CIPDataTypeDINT // Default to DINT
		}

		monitor.AddTag(tag, dataType)

		// Register a callback for value changes
		monitor.OnChange(tag, func(tagName string, value interface{}) {
			fmt.Printf("Tag %s changed to %v\n", tagName, value)
		})
	}

	// Start monitoring
	fmt.Println("Starting tag monitoring...")
	monitor.Start()

	// Run for 30 seconds
	time.Sleep(30 * time.Second)

	// Stop monitoring
	fmt.Println("Stopping tag monitoring...")
	monitor.Stop()

	fmt.Println("Done")
}
