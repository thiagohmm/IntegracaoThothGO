package internal

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

// ExtrairJsonAsync extracts and decompresses a Base64 encoded, compressed JSON string asynchronously
func ExtrairJsonAsync(jsonCompactado string) (map[string]interface{}, error) {
	start := time.Now()

	// Decode Base64 string to bytes
	buffer, err := base64.StdEncoding.DecodeString(jsonCompactado)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 string: %w", err)
	}

	// Decompress the gzip buffer
	gzipReader, err := gzip.NewReader(bytes.NewBuffer(buffer))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	decompressedBytes, err := ioutil.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}

	// Parse JSON from decompressed bytes
	var result map[string]interface{}
	if err := json.Unmarshal(decompressedBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	duration := time.Since(start)
	fmt.Printf("Execution time of ExtrairJsonAsync: %v milliseconds\n", duration.Milliseconds())

	return result, nil
}
