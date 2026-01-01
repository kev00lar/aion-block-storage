package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

const (
	MaxBlockSize = 1 * 1024 * 1024 //1MB Block 
	StorageDir   = "./data/blocks"
	ManifestDir  = "./data/manifests"
)

type Registry struct {
	mu sync.RWMutex
	FileManifest  map[string][]string
	InvertedIndex map[string][]string // keyword ->[]]filenames
}

func main() {
	_ = os.MkdirAll(StorageDir, 0755)
	_ = os.MkdirAll(ManifestDir, 0755)

	reg := &Registry{
		InvertedIndex: make(map[string][]string),
	}

	r := gin.Default()

	r.POST("/upload", reg.HandleUpload)
	r.GET("/search", reg.HandleSearch) // Task 2: Intelligence Endpoint
	r.GET("/download/:filename", reg.HandleDownload)

	fmt.Println("🚀 Document Intelligence System active on :8080")
	r.Run(":8080")
}

func (reg *Registry) HandleSearch(c *gin.Context) {
	query := strings.ToLower(c.Query("q"))
	if query == "" {
		c.JSON(400, gin.H{"error": "Query 'q' is required"})
		return
	}

	reg.mu.RLock()
	results := reg.InvertedIndex[query]
	reg.mu.RUnlock()

	c.JSON(200, gin.H{
		"keyword":  query,
		"found_in": results,
		"count":    len(results),
	})
}

func (reg *Registry) HandleUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("document")
	if err != nil {
		c.JSON(400, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	var blockIDs []string
	buffer := make([]byte, MaxBlockSize)

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			blockID := fmt.Sprintf("%x", sha256.Sum256(chunk))

			// Store Block
			path := filepath.Join(StorageDir, blockID)
			if _, e := os.Stat(path); os.IsNotExist(e) {
				_ = os.WriteFile(path, chunk, 0644)
			}
			blockIDs = append(blockIDs, blockID)

			// Index this block's text immediately
			reg.indexText(header.Filename, chunk)
		}
		if err == io.EOF { break }
	}

	reg.saveManifest(header.Filename, blockIDs)
	c.JSON(200, gin.H{"status": "File indexed and stored", "blocks": len(blockIDs)})
}

func (reg *Registry) HandleDownload(c *gin.Context) {
	filename := c.Param("filename")

	blockIDs, err := reg.loadManifest(filename)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	for _, id := range blockIDs {
		if id == "" { continue }
		blockPath := filepath.Join(StorageDir, id)
		blockData, _ := os.ReadFile(blockPath)
		_, _ = c.Writer.Write(blockData)
	}
}

func (reg *Registry) saveManifest(filename string, blocks []string) {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	path := filepath.Join(ManifestDir, filename+".txt")
	f, _ := os.Create(path)
	defer f.Close()

	for _, id := range blocks {
		fmt.Fprintln(f, id)
	}
}

func (reg *Registry) loadManifest(filename string) ([]string, error) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	path := filepath.Join(ManifestDir, filename+".txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(string(data)), "\n"), nil
}

func (reg *Registry) indexText(filename string, content []byte) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	// function to split by any non-alphanumeric character
	splitter := func(c rune) bool {
		return c == ',' || c == ':' || c == ';' || c == ' ' || c == '\n' || c == '\t' || c == '"' || c == '_'
	}
	words := strings.FieldsFunc(strings.ToLower(string(content)), splitter)
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) < 3 {
			continue
		}

		exists := false
		for _, f := range reg.InvertedIndex[word] {
			if f == filename {
				exists = true
				break
			}
		}

		if !exists {
			reg.InvertedIndex[word] = append(reg.InvertedIndex[word], filename)
		}
	}
}