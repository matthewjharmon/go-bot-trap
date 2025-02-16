package main

import (
	"compress/gzip"
	"embed"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed static/index.html static/script.js static/styles.css static/profile.jpg friendly-words.txt
var embeddedFiles embed.FS

var botTracker = make(map[string]int) // Track bot requests
var mu sync.Mutex
var friendlyWords []string
var mazePaths = make(map[string][]string)
var externalFiles = []string{
	"https://ash-speed.hetzner.com/100MB.bin",
	"https://ash-speed.hetzner.com/1GB.bin",
	"https://ash-speed.hetzner.com/10GB.bin",
}

// Load words from the friendly-words repository
func loadFriendlyWords() []string {
	data, err := embeddedFiles.ReadFile("friendly-words.txt")
	if err != nil {
		log.Fatal("Failed to load words file:", err)
	}
	words := strings.Split(string(data), "\n")
	var cleanWords []string
	for _, word := range words {
		cleaned := strings.TrimSpace(word)
		if cleaned != "" {
			cleanWords = append(cleanWords, cleaned)
		}
	}
	return cleanWords
}

func generateNewPaths(base string) []string {
	var newPaths []string
	for i := 0; i < 5; i++ {
		path := fmt.Sprintf("/m/%s-%d", friendlyWords[rand.Intn(len(friendlyWords))], rand.Intn(10000))
		newPaths = append(newPaths, path)
	}
	mazePaths[base] = newPaths
	return newPaths
}

func init() {
	rand.Seed(time.Now().UnixNano())
	friendlyWords = loadFriendlyWords()
	generateNewPaths("start")
}

// Serve embedded static files for the root route
func staticFileHandler(w http.ResponseWriter, r *http.Request) {
	allowedFiles := map[string]string{
		"/": "static/index.html",
		"/styles.css": "static/styles.css",
		"/script.js": "static/script.js",
		"/profile.jpg": "static/profile.jpg",
	}
	if filePath, exists := allowedFiles[r.URL.Path]; exists {
		data, err := embeddedFiles.ReadFile(filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	} else {
		http.Redirect(w, r, "/m/", http.StatusFound) // Redirect all other routes to the maze
	}
}

// Decrease TCP window size over 100 requests
func getWindowSize(ip string) int {
	mu.Lock()
	botTracker[ip]++
	downloads := botTracker[ip]
	mu.Unlock()

	windowSize := 100 - downloads
	if windowSize < 1 {
		windowSize = 1
	}
	return windowSize
}

// Middleware to enforce TCP window size limits
func windowSizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		windowSize := getWindowSize(r.RemoteAddr)
		w.Header().Set("X-Window-Size", strconv.Itoa(windowSize))
		next.ServeHTTP(w, r)
	})
}

// Generates a large random file with GZIP encoding, applying window size restriction
func generateRandomFile(w http.ResponseWriter, r *http.Request) {
	windowSize := getWindowSize(r.RemoteAddr)
	log.Printf("Serving random file to %s (Window Size: %d)", r.RemoteAddr, windowSize)
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=random-data.gz")

	gzipWriter := gzip.NewWriter(w)
	defer gzipWriter.Close()

	buffer := make([]byte, 1024*windowSize)
	for i := 0; i < (10*1024*1024)/(1024*windowSize); i++ {
		_, err := rand.Read(buffer)
		if err != nil {
			log.Println("Error generating random data:", err)
			return
		}
		_, err = gzipWriter.Write(buffer)
		if err != nil {
			log.Println("Client disconnected from random file")
			return
		}
		gzipWriter.Flush()
	}
}

// Handler to display the maze with new paths and trigger both random and external file downloads
func mazeHandler(w http.ResponseWriter, r *http.Request) {
	windowSize := getWindowSize(r.RemoteAddr)
	path := strings.TrimPrefix(r.URL.Path, "/m/")
	if path == "" {
		path = "start"
	}
	if _, exists := mazePaths[path]; !exists {
		mazePaths[path] = generateNewPaths(path)
	}
	newPaths := mazePaths[path]

	html := "<html><head><title>Welcome to the Maze</title>"
	html += "<style>.hidden-text { display: none; }</style>"
	html += "</head><body>"
	html += fmt.Sprintf("<h1>Welcome to %s (Window Size: %d)</h1>", strings.Title(path), windowSize)
	html += "<p>Explore different paths:</p><ul>"
	for _, p := range newPaths {
		html += fmt.Sprintf("<li><a href='%s'>%s</a></li>", p, strings.Title(strings.TrimPrefix(p, "/m/")))
	}
	html += "</ul><p><a href='/'>Back to Start</a></p>"

	// Randomly insert external download links in the page
	if rand.Intn(3) == 0 {
		html += fmt.Sprintf("<p><a href='%s'>Download Large File</a></p>", externalFiles[rand.Intn(len(externalFiles))])
	}

	// Randomly trigger a download from one of the external files
	if rand.Intn(3) == 0 {
		html += fmt.Sprintf(`<script>setTimeout(function() { window.location.href = "%s"; }, 3000);</script>`, externalFiles[rand.Intn(len(externalFiles))])
	}

	// Add a direct link to trigger the random file download
	html += `<p><a href='/m/random-file'>Download Random Data</a></p>`

	// Automatically trigger the download of the random file after a delay
	html += `<script>setTimeout(function() { window.location.href = '/m/random-file'; }, 5000);</script>`

	html += "</body></html>"

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", staticFileHandler) // Serve index.html, styles, script, profile.jpg
	mux.Handle("/m/", windowSizeMiddleware(http.HandlerFunc(mazeHandler))) // All other routes go to the maze
	mux.Handle("/m/random-file", windowSizeMiddleware(http.HandlerFunc(generateRandomFile))) // Apply window size middleware

	log.Println("Bot maze trap running on 127.0.0.1:8282")
	http.ListenAndServe("127.0.0.1:8282", mux)
}
