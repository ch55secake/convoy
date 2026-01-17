package main

import (
	"log"
	"os"
)

func main() {
	if err := Execute(); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
