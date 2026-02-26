package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chengchuu/go-gin-gee/pkg/logger"
)

// Example: go run scripts/init/main.go -copyData="config.json,database.db"
func main() {
	logger.Println("Init ...")
	// Define command-line flags
	copyData := flag.String("copyData", "config.dev.json,database.db,index.tmpl", "Comma-separated list of files to copy from the assets/data directory")
	flag.Parse()
	targetDir := "data"

	// Copy files from the assets/data directory
	if *copyData != "" {
		// Create the target directory if it doesn't exist
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			if err := os.Mkdir(targetDir, 0755); err != nil {
				logger.Fatal("Error creating directory %s: %v\n", targetDir, err)
			}
		}

		files := strings.Split(*copyData, ",")
		for _, file := range files {
			src := filepath.Join("assets", targetDir, file)
			dst := filepath.Join(targetDir, file)
			if err := copyFile(src, dst); err != nil {
				logger.Printf("Error copying file %s: %v\n", file, err)
			} else {
				logger.Printf("Copied file: %s/%s\n", targetDir, file)
			}
		}
	} else {
		// script.ListFiles("./assets").ExecForEach("cp -R {{.}} .").Stdout()
		logger.Fatal("No files specified to copy. Use -copyData flag.")
	}

	logger.Println("All done.")
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}
