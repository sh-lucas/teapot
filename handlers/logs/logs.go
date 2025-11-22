package logs

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sh-lucas/teapot/cup"
)

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type LogManager struct {
	mu      sync.RWMutex
	writers map[string]chan string
	closed  bool
	wg      sync.WaitGroup
}

var manager = &LogManager{
	writers: make(map[string]chan string),
}

func Shutdown() {
	fmt.Println("Shutting down log manager...")
	manager.mu.Lock()
	if manager.closed {
		manager.mu.Unlock()
		return
	}
	manager.closed = true
	for _, ch := range manager.writers {
		close(ch)
	}
	manager.mu.Unlock()

	manager.wg.Wait()
	fmt.Println("Log manager shutdown complete.")
}

func writerLoop(client string, ch chan string) {
	defer manager.wg.Done()

	filename := fmt.Sprintf("%s.log", client)
	// Open file for appending, create if not exists
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening log file for %s: %v\n", client, err)
		// Consume channel to prevent blocking senders if any (though we are shutting down or erroring)
		for range ch {
		}
		return
	}

	writer := bufio.NewWriter(f)
	ticker := time.NewTicker(1 * time.Second)

	defer func() {
		ticker.Stop()
		fmt.Printf("Flushing and closing log file for %s\n", client)
		writer.Flush()
		f.Close()
	}()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if !strings.HasSuffix(msg, "\n") {
				msg += "\n"
			}
			if _, err := writer.WriteString(msg); err != nil {
				fmt.Printf("Error writing to log buffer for %s: %v\n", client, err)
			}
		case <-ticker.C:
			if err := writer.Flush(); err != nil {
				fmt.Printf("Error flushing log file for %s: %v\n", client, err)
			}
		}
	}
}

// mug:handler POST /log
func SaveLog(w http.ResponseWriter, r *http.Request) {
	writeSecret := cup.WRITE_SECRET
	if writeSecret == "" {
		http.Error(w, "Server configuration error", http.StatusInternalServerError)
		return
	}

	user, pass, ok := r.BasicAuth()
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Username must be WRITE_SECRET
	if user != writeSecret {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Password is the clientName
	clientName := pass
	if clientName == "" {
		http.Error(w, "Client name required", http.StatusBadRequest)
		return
	}

	// Basic sanitization
	if filepath.Base(clientName) != clientName {
		http.Error(w, "Invalid client name", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	manager.mu.RLock()
	if manager.closed {
		manager.mu.RUnlock()
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	ch, exists := manager.writers[clientName]
	if exists {
		// Blocking send to apply backpressure
		ch <- string(body)
		manager.mu.RUnlock()
		w.WriteHeader(http.StatusOK)
		return
	}
	manager.mu.RUnlock()

	// Channel doesn't exist, create it
	manager.mu.Lock()
	// Check again
	if manager.closed {
		manager.mu.Unlock()
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	
	ch, exists = manager.writers[clientName]
	if !exists {
		ch = make(chan string, 100)
		manager.writers[clientName] = ch
		manager.wg.Add(1)
		go writerLoop(clientName, ch)
	}
	
	// Blocking send to apply backpressure
	// If the buffer is full, this will block until the writer catches up.
	// This prevents OOM and ensures no logs are dropped.
	ch <- string(body)
	
	manager.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

// mug:handler GET /logs/{clientName}
func GetLog(w http.ResponseWriter, r *http.Request) {
	readSecret := cup.READ_SECRET
	if readSecret == "" {
		http.Error(w, "Server configuration error", http.StatusInternalServerError)
		return
	}

	authHeader := r.Header.Get("Authorization")
	// Allow "Bearer SECRET" or just "SECRET"
	if authHeader != readSecret && authHeader != "Bearer "+readSecret {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	clientName := chi.URLParam(r, "clientName")
	if clientName == "" {
		http.Error(w, "Client name required", http.StatusBadRequest)
		return
	}

	// Basic sanitization to prevent directory traversal
	if filepath.Base(clientName) != clientName {
		http.Error(w, "Invalid client name", http.StatusBadRequest)
		return
	}

	nStr := r.URL.Query().Get("n")
	n := 50 // Default
	if nStr != "" {
		if val, err := strconv.Atoi(nStr); err == nil && val > 0 {
			n = val
		}
	}

	skipStr := r.URL.Query().Get("skip")
	skip := 0 // Default
	if skipStr != "" {
		if val, err := strconv.Atoi(skipStr); err == nil && val >= 0 {
			skip = val
		}
	}

	filename := fmt.Sprintf("%s.log", clientName)
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Log file not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error opening log file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Ring buffer to store last (n + skip) lines
	bufferSize := n + skip
	ringBuffer := make([]string, 0, bufferSize)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(ringBuffer) < bufferSize {
			ringBuffer = append(ringBuffer, line)
		} else {
			// Shift left and append (or use actual ring logic, but slice append is easier for small n)
			// For strict ring buffer:
			// ringBuffer[writeIdx % bufferSize] = line
			// writeIdx++
			// But then we need to reconstruct order.
			// Since n is likely small (e.g. 50-1000), slice manipulation is fine.
			// Let's use a true ring buffer to avoid O(N) shift.
			// Actually, let's just use slice append/copy for simplicity if n is small.
			// If n is large, this is slow.
			// Let's use a circular buffer approach.
			// But wait, I need to return them in order.
			// Let's just use a slice and append, removing first if full.
			ringBuffer = append(ringBuffer, line)
			ringBuffer = ringBuffer[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		http.Error(w, "Error reading log file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	// We have the last `len(ringBuffer)` lines.
	// We want to skip `skip` from the end.
	// So we take `len(ringBuffer) - skip` lines.
	// And we want at most `n` lines (which should be satisfied by bufferSize = n + skip).

	count := len(ringBuffer) - skip
	if count <= 0 {
		// Nothing to return
		return
	}

	// The lines to return are ringBuffer[0 : count]
	// Because ringBuffer contains the *last* (n+skip) lines.
	// So the ones at the end are the ones to skip.
	// The ones at the beginning are the ones to keep.
	
	for _, line := range ringBuffer[:count] {
		fmt.Fprintln(w, line)
	}
}