package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vitistack/dnsdist-worker/pkg/dnsdist"
)

func main() {
	configFile := flag.String("config", "config.json", "Path to configuration file")
	serverName := flag.String("server", "", "Target server name (leave empty for all servers)")
	command := flag.String("command", "", "Command to execute on dnsdist server(s)")
	listServers := flag.Bool("list", false, "List configured servers")
	flag.Parse()

	// Load configuration
	config, err := dnsdist.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create worker
	worker := dnsdist.NewWorker()
	defer worker.Close()

	// Add servers from configuration
	for _, serverConfig := range config.Servers {
		if err := worker.AddServer(&serverConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error adding server %s: %v\n", serverConfig.Name, err)
			os.Exit(1)
		}
	}

	// Handle list command
	if *listServers {
		servers := worker.GetServerNames()
		fmt.Println("Configured servers:")
		for _, name := range servers {
			fmt.Printf("  - %s\n", name)
		}
		return
	}

	// Validate command
	if *command == "" {
		fmt.Fprintf(os.Stderr, "Error: command is required (use -command flag)\n")
		flag.Usage()
		os.Exit(1)
	}

	// Connect to servers
	if *serverName != "" {
		// Connect to specific server
		if err := worker.Connect(*serverName); err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to %s: %v\n", *serverName, err)
			os.Exit(1)
		}
		fmt.Printf("Connected to %s\n", *serverName)
	} else {
		// Connect to all servers
		errors := worker.ConnectAll()
		if len(errors) > 0 {
			fmt.Println("Connection errors:")
			for name, err := range errors {
				fmt.Printf("  %s: %v\n", name, err)
			}
		}
		connected := worker.GetConnectedServers()
		fmt.Printf("Connected to %d server(s)\n", len(connected))
	}

	// Execute command
	if *serverName != "" {
		// Execute on specific server
		response, err := worker.ExecuteCommand(*serverName, *command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing command on %s: %v\n", *serverName, err)
			os.Exit(1)
		}
		fmt.Printf("Response from %s:\n%s\n", *serverName, response)
	} else {
		// Execute on all servers
		results := worker.ExecuteCommandAll(*command)
		
		// Display results
		fmt.Println("\nResults:")
		for name, result := range results {
			fmt.Printf("\n=== %s ===\n", name)
			if result.Error != nil {
				fmt.Printf("Error: %v\n", result.Error)
			} else {
				fmt.Printf("%s\n", result.Response)
			}
		}

		// Summary
		successCount := 0
		for _, result := range results {
			if result.Error == nil {
				successCount++
			}
		}
		fmt.Printf("\n%d/%d commands executed successfully\n", successCount, len(results))
	}
}
