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

var (
	endpoint_url = "192.168.59.180:2379"
	prefix = ""
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

	fmt.Println("Successfully connected to etcd!")

	//Inserting a key
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = cli.Put(ctx, "key2","value2") //I inserted this key during etcd installation
	cancel()

	if err != nil {
		log.Printf("failed to put key-value pair: %v", err)
	} else {
		fmt.Println("Successfully put key 'mykey' with value 'myvalue'")
	}
	//Let's try to get a key
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := cli.Get(ctx, "key2") //I inserted this key during etcd installation
	cancel()

	if err != nil {
		log.Printf("Failed to get key 'name' : %v", err)
	} else {
		for _, ev := range resp.Kvs {
			fmt.Printf("Key '%s' = '%s'\n", ev.Key, ev.Value)
		}
	}

	// fetching all keys and values...
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	resp, err = cli.Get(ctx, prefix,clientv3.WithPrefix()) //Get prefix : etcdctl get --prefix ""
	cancel()

	if err != nil {
		log.Fatalf("Failed to get keys with prefex: %v",err)
		return
	}

	if len(resp.Kvs) == 0 {
		fmt.Println("No keys found in the etcd database.")
		return
	}

	fmt.Println("--- Keys and Values ---")
	for _, ev := range resp.Kvs {
		fmt.Printf("Key: %s, Value: %s\n", ev.Key, ev.Value)
	}
	fmt.Println("----------------------")
}