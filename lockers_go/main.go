package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.bug.st/serial"
)

// sendSerialCommand remains the same as it contains no web-framework-specific code.
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

// sendSerialCommandWithResponse also remains the same.
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
			log.Printf("      ... read %d bytes in this chunk ...\n", bytesRead)
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
	log.Printf("      Full Response (HEX): % X\n", fullResponse)

	return fullResponse, nil
}

// commandHandler is refactored for Gin.
func commandHandler(c *gin.Context) {
	log.Println("Received API request, attempting to send serial command...")
	err := sendSerialCommand()

	if err != nil {
		log.Printf("--- SERIAL PORT ERROR: %v ---\n", err)
		// Use Gin's JSON response for errors
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send serial command."})
		return
	}

	log.Println("Successfully handled API request and sent 200 OK response.")
	c.String(http.StatusOK, "Command sent successfully.")
}

// commandWithResponseHandler is refactored for Gin.
func commandWithResponseHandler(c *gin.Context) {
	log.Println("Received API request for command with response...")
	response, err := sendSerialCommandWithResponse()

	if err != nil {
		log.Printf("--- SERIAL PORT ERROR: %v ---\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send serial command or get response."})
		return
	}

	log.Println("Successfully handled API request and sent response.")
	// Use Gin's JSON helper to send the response.
	c.JSON(http.StatusOK, gin.H{"response_hex": fmt.Sprintf("%X", response)})
}

func main() {
	// Initialize a new Gin router. `gin.Default()` includes logger and recovery middleware.
	router := gin.Default()

	// --- CONFIGURE CORS for Gin ---
	// This configuration allows all origins, methods, and headers.
	// For production, you should restrict these to known sources.
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// --- DEFINE ROUTES ---
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	// Use router.POST to be more explicit about the required HTTP method.
	router.POST("/send-command", commandHandler)
	router.POST("/send-command-with-response", commandWithResponseHandler)

	const serverAddr = "0.0.0.0:8080"
	log.Printf("Starting Gin web server with CORS enabled on %s\n", serverAddr)

	// Start the server. `router.Run()` is a convenience wrapper around http.ListenAndServe.
	// It will log any errors and Fatal.
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
