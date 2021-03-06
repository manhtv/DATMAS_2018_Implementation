# DATMAS_2018_Implementation

## Install Go:
Version 1.10.1 (1.9+ is required)
```
cd $HOME
curl -O https://dl.google.com/go/go1.10.1.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.10.1.linux-amd64.tar.gz

# Run these two commands and then add them at the end of $HOME/.bashrc
export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
```

## Install latest version of dependencies:
The following instructions were tested with the following versions:
* IPFS 0.4.15-rc1
* IPFS Cluster 0.3.5-a0a08987191b36d610bf1edaf39dd72cd924f0eb
* Tendermint 0.19.2-64408a40

```
go get -d -u github.com/ipfs/go-ipfs
cd $GOPATH/src/github.com/ipfs/go-ipfs
make install
ipfs init

go get -d -u github.com/ipfs/ipfs-cluster
cd $GOPATH/src/github.com/ipfs/ipfs-cluster
make install
ipfs-cluster-service init

go get -u github.com/gorilla/mux

go get github.com/tendermint/tendermint/cmd/tendermint
cd $GOPATH/src/github.com/tendermint/tendermint
make get_tools
make get_vendor_deps
make install
tendermint init

cd $GOPATH/src/google.golang.org/grpc
git reset --hard d11072e7ca9811b1100b80ca0269ac831f06d024

cd $GOPATH/src/github.com/golang/protobuf
git reset --hard 925541529c1fa6821df4e44ce2723319eb2be768

# See: IPFS import conflicts
# See: Multiple registrations 
```
## Install tested version of dependencies:
```
# Warning: You might want to rename or backup your current $GOPATH/src folder.
curl -O www.ux.uis.no/~racin/TestedDependencies.tar.gz
mkdir $GOPATH
tar -C $GOPATH -xzf TestedDependencies.tar.gz 

cd $GOPATH/src/github.com/ipfs/go-ipfs
make install
ipfs init

cd $GOPATH/src/github.com/ipfs/ipfs-cluster
make install
ipfs-cluster-service init

cd $GOPATH/src/github.com/tendermint/tendermint
make install
tendermint init
```

## Install code from this project:
```
cd $GOPATH/src/github.com
mkdir racin
cd racin/
git clone https://github.com/racin/DATMAS_2018_Implementation
cd DATMAS_2018_Implementation/
sh install.sh

go build main.go
go build client/main.go
go build ipfsproxy/main.go
```

## Running 
1. Start tendermint core with "tendermint node"
2. Start tendermint app with "cd $GOPATH/src/github.com/racin/DATMAS_2018_Implementation && ./main"
3. Start IPFS with "ipfs daemon"
4. Start IPFS-cluster with "ipfs-cluster-service"
5. Start IPFS proxy with "cd $GOPATH/src/github.com/racin/DATMAS_2018_Implementation/ipfsproxy && ./main"

## Client Commands
1. Open client directory.
```
cd $GOPATH/src/github.com/racin/DATMAS_2018_Implementation/client
```
### Upload data
_data upload [file] [name] [description]_
```
./main data upload /home/ubuntu/Downloads/Transperth_Sets.JPG TrainImg.jpg
```
If you have access to the gateway, try opening http://[host]:8080/ipfs/[CID]
### Download data
_data get [CID]_
```
./main data get QmYTuLefWfQ7BgER73gRUHA7ZVSJ6Y33JybfpuumdBLG25
```
### Delete data
_data delete [CID]_
```
./main data delete QmYTuLefWfQ7BgER73gRUHA7ZVSJ6Y33JybfpuumdBLG25
```
### Change access on data
_data access [CID] [READERS]_
```
./main data access QmYTuLefWfQ7BgER73gRUHA7ZVSJ6Y33JybfpuumdBLG25 17e25dc5568b5caa0285b04baad76f69,cab4fa317b898a854bb6d2ff2ebead65
```
### Verify storage (Challenge)
_challenge [CID] [challenge] [proof]_
```
./main challenge QmYTuLefWfQ7BgER73gRUHA7ZVSJ6Y33JybfpuumdBLG25
```
### View blockchain
_blockchain view [height]_
```.
./main blockchain view 5
```
### List locally stored metadata
```./main list```
### Get storage node status
_status [CID] [Node]_
```
./main status all d8cabe7f44756a9d6972b539d84a9912
```

## New protobuf:
```
protoc -I=types/ -I=$GOPATH/src -I=$GOPATH/src/github.com/gogo/protobuf/protobuf --go_out=plugins=grpc:types/ types.proto
```

## Common problems:
### IPFS import conflicts:
In the latest versions there is some import conflics with IPFS. See discussion: https://github.com/ipfs/go-ipfs-api/issues/75
A simple fix that works in our case:
Edit: $GOPATH/src/gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto/properties.go
1. Remove log import
2. Function RegisterEnum: Comment out the two panic calls
3. Function RegisterType: Comment out the log call

### Multiple registrations
'http: multiple registrations for /debug/requests'
```
go get -u golang.org/x/net/trace
rm -rf $GOPATH/src/github.com/tendermint/tendermint/vendor/golang.org/x/net/trace
```

### Tendermint configuration
(Installation script will do this).
Make sure $HOME/.tendermint/config/config.toml:
1. abci = "grpc"
2. create_empty_blocks = false

### File not getting pinned (IPFS)
Try restarting the cluster node.

## Generate certificate:
```
mkdir $HOME/.bcfs
cd $HOME/.bcfs
echo -ne '\n' | openssl req -x509 -nodes -newkey rsa:4096 -keyout mycert.pem -out mycert.pem
chmod 0600 mycert.pem
openssl rsa -in mycert.pem -pubout > mycert.pub
```

## Get fingerprint of certificate
### From Private key
```
ssh-keygen -yf mycert.pem > mycert_ssh.pub
ssh-keygen -E md5 -lf mycert_ssh.pub | egrep -o '([0-9a-f]{2}:){15}.{2}' | sed -E 's/://g'
```

### From Public key
```
ssh-keygen -E MD5 -lf /dev/stdin <<< $( ssh-keygen -im PKCS8 -f mycert.pub ) | egrep -o '([0-9a-f]{2}:){15}.{2}' | sed -E 's/://g'
```
