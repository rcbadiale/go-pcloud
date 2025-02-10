package main

import (
	"fmt"
	"io"
	"time"

	pcloud "github.com/rcbadiale/go-pcloud"
)

func main() {
	pc := pcloud.NewPCloud("https://api.pcloud.com", "pcloudToken", nil)
	originalPath := "/example.txt"
	originalFile, err := pcloud.NewFile(&pc, originalPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	randomData := fmt.Sprintf("Hello, world from file %s at %s\n", originalFile.Name, time.Now().Format(time.RFC3339))
	nOriginal, err := originalFile.Write([]byte(randomData))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	copyPath := "/example_copy.txt"
	copyFile, err := pcloud.NewFile(&pc, copyPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	nCopy, err := io.Copy(copyFile, originalFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Original file %s wrote %d bytes\n", originalFile.Name, nOriginal)
	fmt.Printf("Copied file %s wrote %d bytes\n", copyFile.Name, nCopy)
}
