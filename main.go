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
}

func main() {
	// Initialize directories
	_ = os.MkdirAll(StorageDir, 0755)
	_ = os.MkdirAll(ManifestDir, 0755)

	reg := &Registry{}
	r := gin.Default()

	r.POST("/upload", reg.HandleUpload)
	r.GET("/download/:filename", reg.HandleDownload)

	fmt.Println(" Block Storage Service starting on port 8080")
	r.Run(":8080")
}

func (reg *Registry) HandleUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	var blockIDs []string
	buffer := make([]byte, MaxBlockSize) // 1MB buffer"

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			chunk := buffer[:n] // Handles the "Tail Block" fragment

			blockID := fmt.Sprintf("%x", sha256.Sum256(chunk))

			// 2. Persist to Disk (Deduplication)
			savePath := filepath.Join(StorageDir, blockID)
			if _, existsErr := os.Stat(savePath); os.IsNotExist(existsErr) {
				_ = os.WriteFile(savePath, chunk, 0644)
			}

			blockIDs = append(blockIDs, blockID)
		}
		if err == io.EOF {
			break
		}
	}

	reg.saveManifest(header.Filename, blockIDs)

	c.JSON(http.StatusOK, gin.H{
		"filename":    header.Filename,
		"block_count": len(blockIDs),
		"status":      "File shredded into 1MB blocks",
	})
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