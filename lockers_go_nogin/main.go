package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.bug.st/serial"
)

// sendSerialCommand remains the same as it contains no web-framework-specific code.
func sendSerialCommand() error {
	const portName = "/dev/ttyUSB0"
	const baudRate = 9600
	commandToSend := []byte{0xAA, 0x55, 0x03, 0x01, 0x50, 0x00, 0x80}

	log.Printf("--> Preparing to send command to %s\n", portName)
	log.Printf("     Command (HEX): % X\n", commandToSend)

	mode := &serial.Mode{
		BaudRate: baudRate,
	}

	log.Printf("--> Opening port %s...\n", portName)
	port, err := serial.Open(portName, mode)
	if err != nil {
		return fmt.Errorf("failed to open port %s: %v", portName, err)
	}
	defer port.Close()

	log.Println("--> Sending command...")
	bytesWritten, err := port.Write(commandToSend)
	if err != nil {
		return fmt.Errorf("failed to write to serial port: %v", err)
	}

	log.Printf("--> Command sent successfully (%d bytes written).\n", bytesWritten)
	return nil
}

// sendSerialCommandWithResponse also remains the same.
func sendSerialCommandWithResponse() ([]byte, error) {
	const portName = "/dev/ttyUSB0"
	const baudRate = 9600
	commandToSend := []byte{0xAA, 0x55, 0x02, 0x01, 0x51, 0xD9}

	log.Printf("--> Preparing to send command with response to %s\n", portName)
	log.Printf("     Command (HEX): % X\n", commandToSend)

	mode := &serial.Mode{
		BaudRate: baudRate,
	}

	log.Printf("--> Opening port %s...\n", portName)
	port, err := serial.Open(portName, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to open port %s: %v", portName, err)
	}
	defer port.Close()

	err = port.ResetInputBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to reset input buffer: %v", err)
	}
	log.Println("--> Cleared serial input buffer.")

	port.SetReadTimeout(200 * time.Millisecond)

	log.Println("--> Sending command...")
	_, err = port.Write(commandToSend)
	if err != nil {
		return nil, fmt.Errorf("failed to write to serial port: %v", err)
	}
	log.Println("--> Command sent. Now entering read loop for up to 5 seconds...")

	var fullResponse []byte
	var totalBytesRead int
	loopStartTime := time.Now()

	for time.Since(loopStartTime) < 5*time.Second {
		tempBuff := make([]byte, 128)
		bytesRead, err := port.Read(tempBuff)

		if bytesRead > 0 {
			log.Printf("     ... read %d bytes in this chunk ...\n", bytesRead)
			fullResponse = append(fullResponse, tempBuff[:bytesRead]...)
			totalBytesRead += bytesRead
		}

		if err != nil {
			log.Printf("DEBUG: An error occurred. Type: '%T', Value: '%v'", err, err)
			break
		}
	}

	if totalBytesRead == 0 {
		return nil, fmt.Errorf("no response from serial device within 5 seconds")
	}

	log.Printf("--> Response received successfully (total %d bytes read).\n", totalBytesRead)
	log.Printf("     Full Response (HEX): % X\n", fullResponse)

	return fullResponse, nil
}

// commandHandler is refactored for net/http.
func commandHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received API request, attempting to send serial command...")

	// Handle preflight OPTIONS request for CORS
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := sendSerialCommand()
	if err != nil {
		log.Printf("--- SERIAL PORT ERROR: %v ---\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to send serial command."})
		return
	}

	log.Println("Successfully handled API request and sent 200 OK response.")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Command sent successfully."))
}

// commandWithResponseHandler is refactored for net/http.
func commandWithResponseHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received API request for command with response...")

	// Handle preflight OPTIONS request for CORS
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response, err := sendSerialCommandWithResponse()
	if err != nil {
		log.Printf("--- SERIAL PORT ERROR: %v ---\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to send serial command or get response."})
		return
	}

	log.Println("Successfully handled API request and sent response.")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"response_hex": fmt.Sprintf("%X", response)})
}

// corsMiddleware is a simple middleware to handle CORS.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins. In production, restrict this.
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET") // Allow specific methods
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type") // Allow specific headers
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Create a new ServeMux for routing
	mux := http.NewServeMux()

	// Wrap handlers with CORS middleware
	mux.Handle("/ping", corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	})))
	mux.Handle("/send-command", corsMiddleware(http.HandlerFunc(commandHandler)))
	mux.Handle("/send-command-with-response", corsMiddleware(http.HandlerFunc(commandWithResponseHandler)))

	const serverAddr = "0.0.0.0:8080"
	log.Printf("Starting standard Go web server with CORS enabled on %s\n", serverAddr)

	// Start the server
	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}