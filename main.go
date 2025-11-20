package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("ERROR: Do not run 'go run main.go' from the repository root.")
	fmt.Println("")
	fmt.Println("To run the datacat server:")
	fmt.Println("  cd cmd/datacat-server")
	fmt.Println("  go run .")
	fmt.Println("")
	fmt.Println("To run the datacat web UI:")
	fmt.Println("  cd cmd/datacat-web")
	fmt.Println("  go run .")
	fmt.Println("")
	fmt.Println("To build binaries with proper names:")
	fmt.Println("  cd cmd/datacat-server && go build -o datacat-server")
	fmt.Println("  cd cmd/datacat-web && go build -o datacat-web")
	fmt.Println("  cd cmd/datacat-daemon && go build -o datacat-daemon")
	fmt.Println("")
	fmt.Println("Or use the PowerShell scripts in the scripts/ directory:")
	fmt.Println("  .\\scripts\\run-server.ps1")
	fmt.Println("  .\\scripts\\run-web.ps1")
	fmt.Println("")
	os.Exit(1)
}
