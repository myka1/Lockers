package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rs/cors"
	"go.bug.st/serial"
)

// sendSerialCommand remains the same
func sendSerialCommand() error {
	const portName = "COM3"
	const baudRate = 9600
	commandToSend := []byte{0xAA, 0x55, 0x03, 0x01, 0x50, 0x00, 0x80}

	log.Printf("--> Preparing to send command to %s\n", portName)
	log.Printf("      Command (HEX): % X\n", commandToSend)

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

// *** DIAGNOSTIC FUNCTION to identify the timeout error ***
func sendSerialCommandWithResponse() ([]byte, error) {
	const portName = "COM3"
	const baudRate = 9600
	commandToSend := []byte{0xAA, 0x55, 0x02, 0x01, 0x51, 0xD9}

	log.Printf("--> Preparing to send command with response to %s\n", portName)
	log.Printf("      Command (HEX): % X\n", commandToSend)

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
			log.Printf("    ... read %d bytes in this chunk ...\n", bytesRead)
			fullResponse = append(fullResponse, tempBuff[:bytesRead]...)
			totalBytesRead += bytesRead
		}

		if err != nil {
			// =================================================================
			// --- TEMPORARY DEBUGGING CODE ---
			// This line will print the exact type and value of the error.
			log.Printf("DEBUG: An error occurred. Type: '%T', Value: '%v'", err, err)
			// =================================================================

			// For now, we just break on any error.
			break
		}
	}

	if totalBytesRead == 0 {
		return nil, fmt.Errorf("no response from serial device within 5 seconds")
	}

	log.Printf("--> Response received successfully (total %d bytes read).\n", totalBytesRead)
	log.Printf("      Full Response (HEX): % X\n", fullResponse)

	return fullResponse, nil
}

// commandHandler remains the same
func commandHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	log.Println("Received API request, attempting to send serial command...")
	err := sendSerialCommand()

	if err != nil {
		log.Printf("--- SERIAL PORT ERROR: %v ---\n", err)
		http.Error(w, "Failed to send serial command.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Command sent successfully.")
	log.Println("Successfully handled API request and sent 200 OK response.")
}

// *** NEW HANDLER for the new API call ***
func commandWithResponseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	log.Println("Received API request for command with response...")
	response, err := sendSerialCommandWithResponse()

	if err != nil {
		log.Printf("--- SERIAL PORT ERROR: %v ---\n", err)
		http.Error(w, "Failed to send serial command or get response.", http.StatusInternalServerError)
		return
	}

	// Respond with the captured data, encoded in hexadecimal
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"response_hex": "%X"}`, response)
	log.Println("Successfully handled API request and sent response.")
}

func main() {
	mux := http.NewServeMux()

	// Register your handlers on the multiplexer
	mux.HandleFunc("/send-command", commandHandler)
	mux.HandleFunc("/send-command-with-response", commandWithResponseHandler) // <-- Register new handler

	// --- CONFIGURE CORS ---
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://127.0.0.1:5500"},
		AllowedMethods:   []string{http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           int(12 * time.Hour / time.Second),
	})

	handler := c.Handler(mux)

	const serverAddr = ":8080"
	log.Printf("Starting web server with CORS enabled on http://localhost%s\n", serverAddr)

	log.Fatal(http.ListenAndServe(serverAddr, handler))
}
