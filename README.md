# etcd-walker
Playing around with etcd operations
* get-etcd.sh - a script to rapidly kickstart an etcd instance on your box
* etcd-walker.go - a go program to navigate through etcd


### Installation
````
$ mkdir ${PWD}/etcd
````

Copy the the content of the get-etcd.sh script into the newly created directory
````
$ chmod u+x get-etcd.sh
$ ./get-etcd.sh
````
### Copy the binaries to /usr/local/bin
````
$ sudo cp  ${PWD}/etcd/etcd-download-test/etcd /usr/local/bin
$ sudo cp  ${PWD}/etcd/etcd-download-test/etcdctl /usr/local/bin
````

### Test the working
```
$ etcdctl version
etcdctl version: 3.6.0-rc.4
API version: 3.6
```

### Create the service
````
sudo mkdir /etc/etcd
sudo mkdir /var/lib/etcd

sudo groupadd --system etcd
sudo useradd -s /sbin/login --system -g etcd etcd
sudo chown -R etcd:etcd /var/lib/etcd

// Create the service
sudo cat <<EOF >> /etc/systemd/system/etcd.service
[Unit]

Description=etcd service
Documentation=https://github.com/etcd-io/etcd
After=network.target
After=network-online.target
Wants=network-online.target

[Service]

User=etcd
Type=notify
Environment=ETCD_DATA_DIR=/var/lib/etcd
Environment=ETCD_NAME=%m
ExecStart=/usr/local/bin/etcd \
        --listen-client-urls http://0.0.0.0:2379 \
        --advertise-client-urls http://0.0.0.0:2379 \
        --initial-advertise-peer-urls http://0.0.0.0:2379
Restart=always
RestartSec=10s
LimitNOFILE=40000

[Install]
WantedBy=multi-user.target
EOF

//Reload the systemctl daemon
$ sudo systemctl daemon-reload

$ sudo systemctl status etcd.service
$ sudo systemctl start etcd.service


// Testing input
$ etcdctl put name cedric
OK

$ etcdctl get name
name
cedric
````
### Open firewall
````
$ sudo firewall-cmd --add-port={2379/tcp,2380/tcp} --permanent
$ sudo firewall-cmd --reload
````
### View exposure
````
$ ss -tnlp | grep 2379
$ netstat -tnlp | grep 2379
tcp        0      0 127.0.0.1:2379          0.0.0.0:*               LISTEN      -
````

## Setup the go project
### Minimal code

Insert all go imports
````
$ go mod github.com/CedricElie/etcd-walker
$ go tidy 
````

minimal etcd-walkger.go
````
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
			fmt.Printf("Key '%s' = '%s'\n", ev.Key, ev.Value)
		}
	}

	
}
````
### Running
````
$ go build etcd-walker.go
$ go run etcd-walker.go
Successfully connected to etcd!
Key 'name' = 'cedric'
````