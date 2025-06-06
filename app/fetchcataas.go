package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"time" 
)

const externalAPIImageEndpoint = "https://cataas.com/cat" 

// HTML template for the main page
const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>cat vibe checker</title>
    <style>
        body { font-family: sans-serif; text-align: center; }
        img { border: 1px solid #ccc; margin-top: 20px; }
    </style>
</head>
<body>
	<h1>This cat matches your vibe</h1>
    <h2>Not feeling it?</h2>
    <p>Refresh this page to try again</p>
    <!--
        We add a timestamp query parameter to help prevent aggressive browser caching
        of the image URL itself, ensuring our /image-from-api endpoint is hit.
    -->
    <img src="/image-from-api?t={{.Timestamp}}" alt="Dynamic API Image" />
    <p><a href="/">Reload Page</a></p>
</body>
</html>
`

var tmpl = template.Must(template.New("mainPage").Parse(htmlTemplate))

// Data for the HTML template (just for the cache-busting timestamp)
type PageData struct {
	Timestamp string
}

// Handler to serve the main HTML page
func serveHTML(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Timestamp: fmt.Sprintf("%d", time.Now().UnixNano()),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Error executing HTML template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Handler to fetch image from external API and serve it
func serveImageFromAPI(w http.ResponseWriter, r *http.Request) {
	log.Println("Request received for /image-from-api")

	//Call the external API
	resp, err := http.Get(externalAPIImageEndpoint)
	if err != nil {
		log.Printf("Failed to get image from external API: %v", err)
		http.Error(w, "Failed to fetch image from source", http.StatusBadGateway) // 502
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("External API request failed with status code: %d %s", resp.StatusCode, resp.Status)
		// You might want to read the body here for more error details from the API
		http.Error(w, "Image source API returned an error", http.StatusBadGateway)
		return
	}

	//Set the Content-Type header for the image
	//use the Content-Type from the external API's response
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		// Fallback if Content-Type is missing (less ideal)
		// try http.DetectContentType here, but it requires reading some bytes first
		log.Println("Warning: External API did not provide Content-Type. Assuming image/jpeg.")
		contentType = "image/jpeg" // Or a sensible default
	}
	w.Header().Set("Content-Type", contentType)

	// Set cache control headers to indicate the image can change
	// This tells browsers (and proxies) not to cache this response aggressively.
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Stream the image data directly to the client
	// io.Copy is efficient for this.
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Failed to stream image data to client: %v", err)
		// Don't send http.Error here if headers have already been written.
		// The connection might be broken.
	}
	log.Printf("Successfully served image with Content-Type: %s", contentType)
}

func main() {
	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/image-from-api", serveImageFromAPI)


	log.Println("Starting server...")
	// Use a hardcoded port temporarily, later use an env variable
	port := "8080"
	log.Printf("Server starting on http://localhost:%s\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

