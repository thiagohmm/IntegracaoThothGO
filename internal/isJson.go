package internal

import (
	"encoding/json"
	"fmt"
	"regexp"
)

func isJSON(input interface{}) bool {
	var str string

	// Check if the input is not a string, and convert it to a JSON string if necessary
	switch v := input.(type) {
	case string:
		str = v
	default:
		jsonBytes, err := json.Marshal(input)
		if err != nil {
			fmt.Println("Error converting to JSON string:", err)
			return false
		}
		str = string(jsonBytes)
	}

	fmt.Println("Received JSON string:", str)

	// Check for bad escape sequences
	badEscapePattern := regexp.MustCompile(`\\[^"\\bfnrtu]`)
	if badEscapePattern.MatchString(str) {
		fmt.Println("Bad escape sequence detected")
		return false
	}

	// Try to parse the string as JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(str), &js); err != nil {
		fmt.Println("Error parsing JSON string:", err)
		return false
	}

	return true
}
