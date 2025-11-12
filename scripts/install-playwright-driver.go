// +build playwright

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/playwright-community/playwright-go"
)

func main() {
	// Set the environment variables to point to our local playwright directory
	projectRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	playwrightDir := filepath.Join(projectRoot, "..", "bin", "playwright")
	driverDir := filepath.Join(playwrightDir, "playwright-driver")
	browsersDir := filepath.Join(playwrightDir, "browsers")

	// Create directories if they don't exist
	if err := os.MkdirAll(driverDir, 0755); err != nil {
		log.Fatalf("Failed to create driver directory: %v", err)
	}
	if err := os.MkdirAll(browsersDir, 0755); err != nil {
		log.Fatalf("Failed to create browsers directory: %v", err)
	}

	// Set environment variables
	os.Setenv("PLAYWRIGHT_DRIVER_PATH", driverDir)
	os.Setenv("PLAYWRIGHT_BROWSERS_PATH", browsersDir)

	fmt.Printf("Installing Playwright driver to: %s\n", driverDir)
	fmt.Printf("Browser binaries will be installed to: %s\n", browsersDir)

	// Use playwright-go's built-in installation
	err = playwright.Install()
	if err != nil {
		log.Fatalf("Failed to install playwright driver: %v", err)
	}

	fmt.Println("Playwright driver installation completed successfully!")
}