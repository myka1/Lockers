import serial

# =================================================================
# Python Script to Send a Specific, Pre-calculated Hex Command
# =================================================================

# --- USER CONFIGURATION ---
# Change this to your RS485 adapter's COM port
COM_PORT = 'COM3'
BAUD_RATE = 9600

# The exact command you want to send, with each byte represented in hex.
# This is the command: AA 55 03 00 50 00 2B
command_to_send = bytes([0xAA, 0x55, 0x03, 0x01, 0x50, 0x00, 0x80])
#AA 55 03 01 50 00 80

# --- MAIN SCRIPT ---
def main():
    """
    Opens the specified serial port, sends the hardcoded command,
    and then closes the port.
    """
    print(f"--> Preparing to send specific command to {COM_PORT}...")
    print(f"    Command (HEX): {' '.join(f'{b:02X}' for b in command_to_send)}")

    ser = None  # Initialize to None
    try:
        # Open the serial port
        print(f"--> Opening port {COM_PORT}...")
        ser = serial.Serial(COM_PORT, BAUD_RATE, timeout=1.0) # 1-second timeout

        # Send the command
        print("--> Sending command...")
        ser.write(command_to_send)
        print("--> Command sent successfully.")

    except serial.SerialException as e:
        print(f"\n--- SERIAL PORT ERROR ---")
        print(f"Error: {e}")
        print("Please check the following:")
        print(f"1. Is the COM port number '{COM_PORT}' correct?")
        print("2. Is the USB adapter plugged in?")
        print("3. Is another program using the port?")

    finally:
        # Ensure the port is always closed
        if ser and ser.is_open:
            ser.close()
            print("--> Port closed.")


# This ensures the main() function is called when the script is executed
if __name__ == "__main__":
    main()