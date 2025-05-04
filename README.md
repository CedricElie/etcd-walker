# etcd-walker
Playing around with etcd operations
* get-etcd.sh - a script to rapidly kickstart an etcd instance on your box
* etc-walker.go - a go program to navigate through etcd


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
ExecStart=/usr/local/bin/etcd
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

### View exposure
````
$ ss -tnlp | grep 2379
$ netstat -tnlp | grep 2379
tcp        0      0 127.0.0.1:2379          0.0.0.0:*               LISTEN      -
````
