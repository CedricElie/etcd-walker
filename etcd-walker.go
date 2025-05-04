package main

// go mod github.com/CedricElie/etcd-walker
// go tidy

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	cli, err := clientv3.New(clientv3.Config {
		Endpoints:	[]string{"192.168.59.180:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		fmt.Println("Error connecting")
		return
	} 
	
	defer cli.Close()

	fmt.Println("Successfully connected to etcd!")

	//Let's try to get a key
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := cli.Get(ctx, "name") //I inserted this key during etcd installation
	cancel()

	if err != nil {
		log.Printf("Failed to get key 'name' : %v", err)
	} else {
		for _, ev := range resp.Kvs {
			fmt.Printf("Key '%s' = '%s\n", ev.Key, ev.Value)
		}
	}

	
}