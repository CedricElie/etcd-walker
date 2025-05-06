package main

// go mod github.com/CedricElie/etcd-walker
// go tidy

import (
	"context"
	"fmt"
	"log"
	"time"
	"os"
	"flag"

	clientv3 "go.etcd.io/etcd/client/v3"
	"github.com/CedricElie/etcd-walker/config"
)


func main() {

	//Control the number of parameters sent
	if len(os.Args) <= 1 {
		fmt.Println("Usage: etcd-walker [-ls | -cp | -grep | ...] ")
		os.Exit(1)
	}
	// Define the flags and their associated variables
	var (
		lsFlag      string
		cpSource    string
		grepPattern string
		outputPath  string
	)

	flag.StringVar(&lsFlag, "ls", "", "List etcds")
	flag.StringVar(&cpSource, "cp", "", "Copy source etcd")
	flag.StringVar(&grepPattern, "grep", "", "Search for pattern")
	flag.StringVar(&outputPath, "out", "default.log", "Output etcd path")

	// Parse the command-line arguments
	flag.Parse()

	// Basic validation: Ensure at least one primary action is requested
	if lsFlag == "" && cpSource == "" && grepPattern == "" {
		fmt.Println("Error: Must provide one of ls, cp <source>, or grep <pattern>")
		flag.Usage()
		os.Exit(1)
	}

	// More specific validation for arguments requiring values
	if flag.Lookup("cp").Value.String() != "" && cpSource == "" {
		fmt.Println("Error: The --cp command requires a source etcd value.")
		flag.Usage()
		os.Exit(1)
	}

	if flag.Lookup("grep").Value.String() != "" && grepPattern == "" {
		fmt.Println("Error: The --grep command requires a search pattern value.")
		flag.Usage()
		os.Exit(1)
	}

	// If all controls are OK, let's Connect to etcd
	cfg := config.GetConfig()

	cli, err := clientv3.New(clientv3.Config {
		Endpoints:	[]string{cfg.ETCD_HOST},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		fmt.Println("Error connecting")
		return
	} 
	
	defer cli.Close()

	fmt.Println("======== Successfully connected to etcd! =======\n  ")

	// Process the arguments and their values
	//Implement the functions ls,cp,grep
	if lsFlag != "" {
		lsFlagVal := flag.Lookup("ls").Value.String()
		fmt.Printf("Listing etcd key '%v'\n",lsFlagVal)

		//fmt.Println("Value passed to ls",lsFlagVal)
		// Do the rest here
		//Let's try to get a key
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := cli.Get(ctx, lsFlagVal,clientv3.WithPrefix())
		cancel()

		if err != nil {
			log.Printf("Failed to get key 'name' : %v", err)
		} else {
			for _, ev := range resp.Kvs {
				fmt.Printf("Key '%s', Value = '%s'\n", ev.Key, ev.Value)
			}
		}
		
	}

	if cpSource != "" {
		fmt.Printf("Copying etcd: %s to ... \n", cpSource)
		// 	// Do the rest here for --cp with cpSource value
	}

	if grepPattern != "" {
		fmt.Printf("Searching for pattern: '%s' in ... \n", grepPattern)
		// 	// Do the rest here --grep with grepPattern value
	}

	//fmt.Printf("Output will be written to: %s\n", outputPath)
	
	/*
		//Inserting a key
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = cli.Put(ctx, "key2","value2") //I inserted this key during etcd installation
		cancel()

		if err != nil {
			log.Printf("failed to put key-value pair: %v", err)
		} else {
			fmt.Println("Successfully put key 'mykey' with value 'myvalue'")
		}

		// fetching all keys and values...
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		resp, err = cli.Get(ctx, "/registry",clientv3.WithPrefix()) //Get prefix : etcdctl get --prefix ""
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
	*/
}