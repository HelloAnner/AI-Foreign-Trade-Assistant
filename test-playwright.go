// +build playwright

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/playwright-community/playwright-go"
)

func main() {
	fmt.Println("Testing playwright-go with custom driver path...")
	fmt.Printf("PLAYWRIGHT_DRIVER_PATH: %s\n", os.Getenv("PLAYWRIGHT_DRIVER_PATH"))
	fmt.Printf("PLAYWRIGHT_BROWSERS_PATH: %s\n", os.Getenv("PLAYWRIGHT_BROWSERS_PATH"))

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Failed to start Playwright: %v", err)
	}
	defer pw.Stop()

	fmt.Println("Success: Playwright started with custom driver path!")
}