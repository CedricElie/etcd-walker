package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.etcd.io/etcd/client/v3"
)

var (
	endpoint_url = "192.168.59.180:2379"
	prefix = ""
	filePath = "test/data.etcd"
)

func main() {
	cli, err := clientv3.New(clientv3.Config {
		Endpoints:	[]string{endpoint_url},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		fmt.Println("Error connecting")
		return
	} 
	
	defer cli.Close()

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") { // Skip empty lines and comments
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := cli.Put(ctx, key, value)
			cancel()
			if err != nil {
				log.Printf("failed to put key '%s': %v", key, err)
			} else {
				fmt.Printf("Successfully put key '%s'\n", key)
			}
		} else {
			log.Printf("invalid line format: %s", line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	fmt.Println("Finished inserting data from file.")
}

