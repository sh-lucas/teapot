package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Config holds the client configuration
type Config struct {
	Secret   string
	Host     string
	Command  string
	Args     []string
	Insecure bool // For testing (allows HTTP)
}

func parseArgs() Config {
	cfg := Config{}
	args := os.Args[1:]

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			switch arg {
			case "-s":
				if i+1 < len(args) {
					cfg.Secret = args[i+1]
					i++
				}
			case "-h":
				if i+1 < len(args) {
					cfg.Host = args[i+1]
					i++
				}
			case "--insecure":
				cfg.Insecure = true
			default:
				// Unknown flag, assume it's the command start if it doesn't look like our flag?
				// Requirement: "recieve the first argument as the name of the command to run"
				// But also "recieve flags".
				// Usually flags come before the command.
				// If we hit something that isn't a known flag, we treat it as the command.
				cfg.Command = arg
				cfg.Args = args[i+1:]
				return cfg
			}
		} else {
			// Not a flag, must be the command
			cfg.Command = arg
			cfg.Args = args[i+1:]
			return cfg
		}
	}
	return cfg
}

func main() {
	cfg := parseArgs()

	if cfg.Command == "" {
		fmt.Println("Usage: teapot -s <secret> -h <host> <command> [args...]")
		os.Exit(1)
	}

	if cfg.Host == "" {
		fmt.Println("Host is required (-h)")
		os.Exit(1)
	}

	// Host normalization
	if !strings.HasPrefix(cfg.Host, "http://") && !strings.HasPrefix(cfg.Host, "https://") {
		cfg.Host = "https://" + cfg.Host
	}

	if strings.HasPrefix(cfg.Host, "http://") && !cfg.Insecure {
		fmt.Println("Error: HTTP is blocked. Use HTTPS or --insecure for testing.")
		os.Exit(1)
	}

	// Log buffer
	var (
		mu     sync.Mutex
		buffer bytes.Buffer
	)

	// Start the command
	cmd := exec.Command(cfg.Command, cfg.Args...)
	
	// Capture stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating stdout pipe: %v\n", err)
		os.Exit(1)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating stderr pipe: %v\n", err)
		os.Exit(1)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting command: %v\n", err)
		os.Exit(1)
	}

	// MultiWriter to print to terminal and buffer
	// We need to read from pipes and write to (os.Stdout + buffer)
	
	var wg sync.WaitGroup
	wg.Add(2)

	// Stream stdout
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line) // Print to terminal
			
			mu.Lock()
			buffer.WriteString(line + "\n")
			mu.Unlock()
		}
	}()

	// Stream stderr
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintln(os.Stderr, line) // Print to terminal stderr
			
			mu.Lock()
			buffer.WriteString(line + "\n")
			mu.Unlock()
		}
	}()

	// Ticker for sending logs
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		client := &http.Client{Timeout: 5 * time.Second}

		for {
			select {
			case <-done:
				// Final flush
				flushLogs(client, cfg, &mu, &buffer)
				return
			case <-ticker.C:
				flushLogs(client, cfg, &mu, &buffer)
			}
		}
	}()

	// Wait for command to finish
	err = cmd.Wait()
	wg.Wait() // Wait for output streams to finish
	
	close(done) // Stop ticker and trigger final flush
	// Give a moment for final flush to complete (the goroutine will exit after flush)
	// Actually, we should wait for the logger goroutine to finish.
	// But for simplicity, we can just sleep a tiny bit or use another WG.
	time.Sleep(100 * time.Millisecond) 

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Command execution error: %v\n", err)
		os.Exit(1)
	}
}

func flushLogs(client *http.Client, cfg Config, mu *sync.Mutex, buffer *bytes.Buffer) {
	mu.Lock()
	if buffer.Len() == 0 {
		mu.Unlock()
		return
	}
	data := buffer.Bytes()
	buffer.Reset()
	mu.Unlock()

	req, err := http.NewRequest("POST", cfg.Host+"/log", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log request: %v\n", err)
		return
	}

	req.SetBasicAuth(cfg.Secret, cfg.Command)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending logs: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error sending logs: server returned %s\n", resp.Status)
	}
}
