package main

import (
	"fmt"

	pcloud "github.com/rcbadiale/go-pcloud"
)

func main() {
	pc := pcloud.NewPCloud("https://api.pcloud.com", "pcloudToken", nil)
	fullPath := "/example.txt"
	f, err := pcloud.NewFile(&pc, fullPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("File: %s\n", f.Name)
	n, err := f.Write([]byte("Hello, World!"))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Wrote %d bytes\n", n)
}
