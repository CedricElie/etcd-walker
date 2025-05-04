ETCD_VER=v3.6.0-rc.4

# choose either URL
GOOGLE_URL=https://storage.googleapis.com/etcd
GITHUB_URL=https://github.com/etcd-io/etcd/releases/download
DOWNLOAD_URL=${GOOGLE_URL}

rm -f /${PWD}/etcd-${ETCD_VER}-linux-amd64.tar.gz
rm -rf /${PWD}/etcd-download-test && mkdir -p /${PWD}/etcd-download-test

curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o /${PWD}/etcd-${ETCD_VER}-linux-amd64.tar.gz
tar xzvf /${PWD}/etcd-${ETCD_VER}-linux-amd64.tar.gz -C /${PWD}/etcd-download-test --strip-components=1
rm -f /${PWD}/etcd-${ETCD_VER}-linux-amd64.tar.gz

/${PWD}/etcd-download-test/etcd --version
/${PWD}/etcd-download-test/etcdctl version
/${PWD}/etcd-download-test/etcdutl version

# start a local etcd server
/${PWD}/etcd-download-test/etcd

# write,read to etcd
/${PWD}/etcd-download-test/etcdctl --endpoints=localhost:2379 put foo bar
/${PWD}/etcd-download-test/etcdctl --endpoints=localhost:2379 get foo
