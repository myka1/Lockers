import { Pressable, Text, View } from "react-native";

export default function Index() {
  // Define the endpoint URL inside the component or keep it outside as a constant.
  const openLockerUrl = "http://192.168.88.4:8080/send-command";

  // --- Move the handler function INSIDE the component ---
  const openLocker = async () => {
    console.log("Attempting to open locker with URL:", openLockerUrl);
    try {
      const response = await fetch(openLockerUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (!response.ok) {
        console.error(`HTTP error! Status: ${response.status}`);
        const errorText = await response.text();
        console.error("Error response body:", errorText);
        return;
      }

      const data = await response.text();
      console.log("Server response:", data);
    } catch (error) {
      // --- MODIFIED PART ---
      console.error("Fetch failed. Full error object:");
      console.log(error); // Log the raw error object
      // Also log it as a JSON string to see all properties
      console.error(JSON.stringify(error, null, 2));
    }
  };

  return (
    <View
      style={{
        flex: 1,
        justifyContent: "center",
        alignItems: "center",
      }}
    >
      <View>
        <Pressable onPress={openLocker}>
          {/* A little bit of styling makes it look more like a button */}
          <Text style={{ fontSize: 20, padding: 10, backgroundColor: "#ddd" }}>
            Open locker
          </Text>
        </Pressable>
      </View>
    </View>
  );
}
