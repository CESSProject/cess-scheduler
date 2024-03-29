# <h1 align="center">CESS-SCHEDULER &middot; [![GitHub license](https://img.shields.io/badge/license-Apache2-blue)](#LICENSE) <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.19-blue.svg" /></a> [![Go Reference](https://pkg.go.dev/badge/github.com/CESSProject/cess-scheduler.svg)](https://pkg.go.dev/github.com/CESSProject/cess-scheduler)</h1>

CESS-Scheduler is a scheduling service for consensus miners.

## Reporting a Vulnerability

If you find out any vulnerability, Please send an email to tech@cess.one, we are happy to communicate with you.

## System Requirements

- Linux-amd64

## System dependencies

**Step 1:** Install common libraries

Take the ubuntu distribution as an example:

```shell
sudo apt update && sudo apt upgrade
sudo apt install m4 g++ flex bison make gcc git curl wget lzip vim util-linux -y
```

**Step 2:** Install the necessary pbc library

```shell
sudo wget https://gmplib.org/download/gmp/gmp-6.2.1.tar.lz
sudo lzip -d gmp-6.2.1.tar.lz
sudo tar -xvf gmp-6.2.1.tar
cd gmp-6.2.1/
sudo chmod +x ./configure
sudo ./configure --enable-cxx
sudo make
sudo make check
sudo make install
cd ..

sudo wget https://crypto.stanford.edu/pbc/files/pbc-0.5.14.tar.gz
sudo tar -zxvf pbc-0.5.14.tar.gz
cd pbc-0.5.14/
sudo chmod +x ./configure
sudo ./configure
sudo make
sudo make install
sudo touch /etc/ld.so.conf.d/libpbc.conf
sudo echo "/usr/local/lib" >> /etc/ld.so.conf.d/libpbc.conf
sudo ldconfig
```

## System configuration

- Firewall

If the firewall is turned on, you need to open the running port, the default port is 15000.

Take the ubuntu distribution as an example:

```shell
sudo ufw allow 15001/tcp
```
- Network optimization (optional)

```shell
sysctl -w net.ipv4.tcp_syncookies = 1
sysctl -w net.ipv4.tcp_tw_reuse = 1
sysctl -w net.ipv4.tcp_tw_recycle = 1
sysctl -w net.ipv4.tcp_fin_timeout = 30
sysctl -w net.ipv4.tcp_max_syn_backlog = 8192
sysctl -w net.ipv4.tcp_max_tw_buckets = 6000
sysctl -w net.ipv4.tcp_timestsmps = 0
sysctl -w net.ipv4.ip_local_port_range = 10000 65500
```

## Build from source

**Step 1:** Install go locale

CESS-Bucket requires [Go 1.19](https://golang.org/dl/) or higher.

> See the [official Golang installation instructions](https://golang.org/doc/install) If you get stuck in the following process.

- Download go1.19 compress the package and extract it to the /use/local directory:

```shell
sudo wget -c https://golang.org/dl/go1.19.linux-amd64.tar.gz -O - | sudo tar -xz -C /usr/local
```

- You'll need to add `/usr/local/go/bin` to your path. For most Linux distributions you can run something like:

```shell
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc && source ~/.bashrc
```

- View your go version:

```shell
go version
```

**Step 2:** Build a scheduler

```shell
git clone https://github.com/CESSProject/cess-scheduler.git
cd cess-scheduler/
go build -o scheduler cmd/main.go
```

If all goes well, you will get a mining program called `scheduler`.

## Get started with bucket

**Step 1:** Register two polka wallet

For wallet one, it is called an  `stash account`, which is used as a mortgage to become a scheduler,you need to recharge it at least 2000000TCESS.

For wallet two, it is called a `controller account`, which is used to execute transactions. You need to recharge the account with a small tokens and provide the private key to the scheduler's configuration file. The cess system will not record and destroy the account.

Browser access: [App](https://testnet.cess.cloud/#/explorer) implemented by [CESS Explorer](https://github.com/CESSProject/cess-explorer), [Add two accounts](https://testnet.cess.cloud/#/accounts) in two steps.

**Step 2:** Recharge your signature account

If you are using the test network, Please join the [CESS discord](https://discord.gg/mYHTMfBwNS) to get it for free. If you are using the official network, please buy CESS tokens.

**Step 3:** Become a consensus miner
Browser access: [Stash](https://testnet.cess.cloud/#/staking/actions), click `+Stash` to become consensus.

**Step 4:** Prepare configuration file

Use `scheduler` to generate configuration file templates directly in the current directory:

```shell
sudo chmod +x scheduler
./scheduler default
```

The content of the configuration file template is as follows. You need to fill in your own information into the file. By default, the `bucket` uses `conf.toml` in the current directory as the runtime configuration file. You can use `-c` or `--config` to specify the configuration file Location.
```
# The rpc address of the chain node
RpcAddr     = ""
# The IP address of the machine's public network used by the scheduler program
ServiceAddr = ""
# Port number monitored by the scheduler program
ServicePort = ""
# Data storage directory
DataDir     = ""
# Phrase or seed of the controller account
CtrlPrk     = ""
# The address of stash account
StashAcc    = ""
```
*Our testnet rpc address is as follows:*<br>
`wss://testnet-rpc0.cess.cloud/ws/`<br>
`wss://testnet-rpc1.cess.cloud/ws/`

**Step 5:** View scheduler features

The `scheduler` has many functions, you can use `-h` or `--help` to view, as follows:

- flag

| Flag        | Description                             |
| ----------- | --------------------------------------- |
| -c,--config | Specify the configuration file          |
| -h,--help   | help for cess-scheduler                 |

- command

| Command  | Description                                    |
| -------- | ---------------------------------------------- |
| version  | Print version number                           |
| default  | Generate configuration file template           |
| run      | Register and run the scheduler program         |
| update   | Update scheduling service ip and port          |

**Step 6:** Start scheduler

```shell
sudo ./scheduler run 2>&1 &
```

## Important feature 1: File redundancy
After the scheduler receives the files uploaded by the user, it performs redundant processing on it. The algorithm used is Reed-Solomon, and the redundancy is 0.5 times, see: https://github.com/CESSProject/cess-scheduler/blob/8829362237260239a80d525ecabc0ea94a54ac9e/pkg/erasure/reedsolomon.go#L38-L156

## Important feature 2: File storage
Finally, the scheduling service selects miners in the entire network, and randomly stores files on these miners. Before storing, the miners' certification space will be judged. For details, see: https://github.com/CESSProject/cess-scheduler/blob/8829362237260239a80d525ecabc0ea94a54ac9e/node/handler.go#L399-L607

## Important feature 3: Proof of verification
The scheduling service starts a scheduled task to obtain the proof to be verified. After obtaining the proof, each proof is verified, and some parameter information needs to be obtained from the corresponding miner to complete the verification. If the miner is offline at this time, the proof is reachable. Verification failed, see: https://github.com/CESSProject/cess-scheduler/blob/8829362237260239a80d525ecabc0ea94a54ac9e/node/sub_verifyProofs.go#L33-L130
