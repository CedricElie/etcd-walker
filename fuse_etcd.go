package main

import (
	"bufio"
	"context"
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

// EtcdFS represents the etcd-backed filesystem.
type EtcdFS struct {
	mu   sync.RWMutex
	data map[string]string // Key: etcd key (without the colon), Value: JSON string
}

// Dir represents a directory in the filesystem.
type Dir struct {
	fs   *EtcdFS
	path string
}

// File represents a file containing JSON data.
type File struct {
	fs   *EtcdFS
	path string
}

func (f *EtcdFS) Root() (fs.Node, error) {
	return &Dir{fs: f, path: ""}, nil
}

// Attr for Dir
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = hash(d.path) // Use path for inode
	a.Mode = os.ModeDir | 0o777
	a.Mtime = time.Now()
	a.Ctime = time.Now()
	return nil
}

// Lookup for Dir
func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
    d.fs.mu.RLock()
    defer d.fs.mu.RUnlock()

    fullPath := filepath.Join(d.path, name)
    foundDir := false

    for key := range d.fs.data {
        lookupKey := key
        if d.path == "" && strings.HasPrefix(key, "/") {
            lookupKey = key[1:] // Remove leading slash for root lookup
        }
        if strings.HasPrefix(lookupKey, name+"/") {
            foundDir = true
            break
        }
        if lookupKey == name {
            return &File{fs: d.fs, path: key}, nil
        }
    }

    if foundDir {
        return &Dir{fs: d.fs, path: fullPath}, nil
    }

    return nil, syscall.ENOENT
}

// ReadDirAll for Dir
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	d.fs.mu.RLock()
	defer d.fs.mu.RUnlock()

	var entries []fuse.Dirent
	seen := make(map[string]bool)

	log.Printf("ReadDirAll called for path: %s", d.path)

	for key := range d.fs.data {
		log.Printf("Processing key: %s", key) // ADD THIS LINE
		if strings.HasPrefix(key, d.path) {
			relativePath := strings.TrimPrefix(key, d.path)
			if relativePath != "" && relativePath[0] == '/' {
				relativePath = relativePath[1:]
			}
			parts := strings.SplitN(relativePath, "/", 2)
			name := parts[0]

			if len(parts) == 1 && key == filepath.Join(d.path, name) {
				if !seen[name] {
					log.Printf("Found file: %s", name)
					entries = append(entries, fuse.Dirent{Name: name, Type: fuse.DT_File})
					seen[name] = true
				}
			} else if len(parts) > 1 {
				if !seen[name] {
					log.Printf("Found subdir: %s", name)
					entries = append(entries, fuse.Dirent{Name: name, Type: fuse.DT_Dir})
					seen[name] = true
				}
			}
		}
	}

	return entries, nil
}

// Attr for File
func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	f.fs.mu.RLock()
	defer f.fs.mu.RUnlock()

	if content, ok := f.fs.data[f.path]; ok {
		a.Inode = hash(f.path)
		a.Mode = 0o777
		a.Size = uint64(len(content))
		a.Mtime = time.Now()
		a.Ctime = time.Now()
		return nil
	}
	return syscall.ENOENT
}

// Open for File
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	return f, nil
}

// ReadAll for File (returns the JSON content)
func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	f.fs.mu.RLock()
	defer f.fs.mu.RUnlock()

	if content, ok := f.fs.data[f.path]; ok {
		return []byte(content), nil
	}
	return nil, syscall.ENOENT
}

func loadEtcdData(filePath string) (map[string]string, error) {
	data := make(map[string]string)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open data file: %w", err)
	}
	defer file.Close()

	scanner := NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			data[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read data file: %w", err)
	}

	return data, nil
}

// Scanner is a simple scanner that reads line by line.
type Scanner struct {
	file    *os.File
	scanner *bufio.Scanner
}

// NewScanner creates a new Scanner.
func NewScanner(file *os.File) *Scanner {
	return &Scanner{
		file:    file,
		scanner: bufio.NewScanner(file),
	}
}

// Scan advances the scanner to the next line.
func (s *Scanner) Scan() bool {
	return s.scanner.Scan()
}

// Text returns the current line.
func (s *Scanner) Text() string {
	return s.scanner.Text()
}

// Err returns the first non-EOF error that was encountered by the Scanner.
func (s *Scanner) Err() error {
	return s.scanner.Err()
}

// Simple hash function for inode generation.
func hash(s string) uint64 {
	var h uint64 = 5381
	for i := 0; i < len(s); i++ {
		h = (h << 5) + h + uint64(s[i])
	}
	return h
}

func main() {
	dataPath := flag.String("data", "", "Path to the etcd data file")
	mountPoint := flag.String("mount", "", "Mount point for the filesystem")
	flag.Parse()

	if *dataPath == "" || *mountPoint == "" {
		fmt.Println("Usage: go run etcdfs.go --data <data_file> --mount <mount_point>")
		os.Exit(1)
	}

	data, err := loadEtcdData(*dataPath)
	if err != nil {
		log.Fatalf("Failed to load etcd data: %v", err)
	}

	fsys := &EtcdFS{data: data}

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

	log.Println("FUSE filesystem unmounted.")
}