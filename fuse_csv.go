package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// CSVFS represents the CSV-backed filesystem.
type CSVFS struct {
	mu   sync.RWMutex
	data map[string][]string
}

// Dir represents a directory in the filesystem.
type Dir struct {
	fs   *CSVFS
	path string
}

// File represents an empty file in the filesystem.
type File struct {
	fs   *CSVFS
	path string
}

func (f *CSVFS) Root() (fs.Node, error) {
	return &Dir{fs: f, path: ""}, nil
}

// Attr handles the getattr operation.
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1 // Root inode
	a.Mode = os.ModeDir | 0o555
	a.Mtime = time.Now()
	a.Ctime = time.Now()
	return nil
}

// Lookup handles looking up a directory entry.
func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	d.fs.mu.RLock()
	defer d.fs.mu.RUnlock()

	fullPath := filepath.Join(d.path, name)

	// Check if it's a top-level directory from the CSV keys
	if parentPath(fullPath) == "" {
		if _, ok := d.fs.data[name]; ok {
			return &Dir{fs: d.fs, path: name}, nil
		}
	} else {
		// Check if it's a file within a directory
		parent := parentPath(fullPath)
		if files, ok := d.fs.data[parent]; ok {
			if contains(files, name) {
				return &File{fs: d.fs, path: fullPath}, nil
			}
		}
	}
	return nil, syscall.ENOENT
}

// ReadDirAll handles reading directory entries.
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	d.fs.mu.RLock()
	defer d.fs.mu.RUnlock()

	var entries []fuse.Dirent

	if d.path == "" {
		// Top-level directories (CSV keys)
		for folder := range d.fs.data {
			entries = append(entries, fuse.Dirent{Name: folder, Type: fuse.DT_Dir})
		}
	} else if files, ok := d.fs.data[d.path]; ok {
		// Files within a directory (CSV values)
		for _, file := range files {
			entries = append(entries, fuse.Dirent{Name: file, Type: fuse.DT_File})
		}
	}

	return entries, nil
}

// Attr for File
func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = hash(f.path) // Simple way to get a unique inode
	a.Mode = 0o444
	a.Size = 0
	a.Mtime = time.Now()
	a.Ctime = time.Now()
	return nil
}

// Open for File
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	return f, nil
}

// ReadAll for File (returns empty content)
func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte{}, nil
}

func loadCSV(csvPath string) (map[string][]string, error) {
	data := make(map[string][]string)
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	for _, row := range records {
		if len(row) == 2 {
			folder, filename := row[0], row[1]
			data[folder] = append(data[folder], filename)
		}
	}
	return data, nil
}

func parentPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], "/")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Simple hash function for inode generation. Not cryptographically secure, but sufficient for this example.
func hash(s string) uint64 {
	var h uint64 = 5381
	for i := 0; i < len(s); i++ {
		h = (h << 5) + h + uint64(s[i])
	}
	return h
}
func main() {
	csvPath := flag.String("csv", "", "Path to the CSV file")
	mountPoint := flag.String("mount", "", "Mount point for the filesystem")
	flag.Parse()

	if *csvPath == "" || *mountPoint == "" {
		fmt.Println("Usage: go run csvfs.go --csv <csv_file> --mount <mount_point>")
		os.Exit(1)
	}

	data, err := loadCSV(*csvPath)
	if err != nil {
		log.Fatalf("Failed to load CSV: %v", err)
	}

	fsys := &CSVFS{data: data}

	c, err := fuse.Mount(*mountPoint)
	if err != nil {
		log.Fatalf("Failed to mount FUSE filesystem: %v", err)
	}
	defer c.Close()

	log.Println("FUSE filesystem mounted successfully!")

	err = fs.Serve(c, fsys)
	if err != nil {
		log.Fatalf("Failed to serve FUSE filesystem: %v", err)
	}

	// No need to explicitly check c.Ready and c.MountError here anymore.
	// The fs.Serve function will block until unmounted or an error occurs.
	log.Println("FUSE filesystem unmounted.")
}