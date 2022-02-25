package rotation

import (
	"context"
	"fmt"
	"os"
	"scheduler-mining/configs"
	ce "scheduler-mining/internal/chain"
	"scheduler-mining/internal/logger"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
)

func EtcdClusterRegister() {
	time.Sleep(time.Second * 3)

	logger.InfoLogger.Sugar().Infof("Start Etcd Register")
	commandetcd, role := EtcdStartCommand()
	CleanContainer()
	fmt.Println(commandetcd)
	logger.InfoLogger.Sugar().Infof("This Identity:%v,Allow Participate in Cluster:%v,Perr Info:%v", role.Role, role.AllowJoin, role.Participants)
	switch role.PeerInfo.Role {
	case "genesis":
		exec_shell_immediately(commandetcd)
		if !CommandPid() {
			logger.ErrLogger.Sugar().Errorf("Start etcd fail!!")
			panic("Start etcd fail!!")
		} else {
			logger.InfoLogger.Sugar().Infof("etcd start success!!!!")
			fmt.Println("etcd start success!!!!")
		}
		//etcd pool open
		err := OpenBalance()
		if err != nil {
			fmt.Println("OpenBalance err:", err)
			checkErr(err)
		} else {
			fmt.Println("OpenBalance With No Auth successful")
		}
		_, err = BalanceCon.Client.UserAdd(context.Background(), configs.Confile.RotationModule.Username, configs.Confile.RotationModule.Password)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Genesis Node Add User Error: %v", err)
			checkErr(err)
		}
		fmt.Println("Genesis Node Add User Successful!")
		logger.InfoLogger.Sugar().Infof("Genesis Node Add User Successful!")

		_, err = BalanceCon.Client.RoleAdd(context.Background(), "normal")
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Genesis Node Add Role 'normal' Error: %v", err)
			checkErr(err)
		}
		fmt.Println("Genesis Node Add Role 'normal' Successful!")
		logger.InfoLogger.Sugar().Infof("Genesis Node Add Role 'normal' Successful!")

		_, err = BalanceCon.Client.RoleGrantPermission(context.Background(), "normal", "R", "Rz", 0)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Genesis Node Grant Role Permission With prefix=R Error: %v", err)
			checkErr(err)
		}
		fmt.Println("Genesis Node Grant Role Permission With prefix=R Successful!")
		logger.InfoLogger.Sugar().Infof("Genesis Node Grant Role Permission With prefix=R Successful!")

		_, err = BalanceCon.Client.UserGrantRole(context.Background(), "root", "root")
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Genesis Node Grant Role:root To User:root Error: %v", err)
			checkErr(err)
		}
		fmt.Println("Genesis Node Grant Role:root To User:root Successful!")
		logger.InfoLogger.Sugar().Infof("Genesis Node Grant Role:root To User:root Successful!")

		_, err = BalanceCon.Client.AuthEnable(context.Background())
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Genesis Node Set Cluster Auth Enable Error: %v", err)
			checkErr(err)
		}
		fmt.Println("Genesis Node Set Cluster Auth Enable Successful!")
		logger.InfoLogger.Sugar().Infof("Genesis Node Set Cluster Auth Enable Successful!")

		err = OpenAuthBalance()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Open Etcd Balance With Auth Error: %v", err)
			checkErr(err)
		}
		fmt.Println("Open Etcd Balance With Auth Successful!")
		logger.InfoLogger.Sugar().Infof("Open Etcd Balance With Auth Successful!")

		ipkey := strings.Split(configs.Confile.RotationModule.Endpoint, ":")
		BalanceCon.Client.Put(context.Background(), "/"+ipkey[0], "root")
		BalanceCon.Client.Put(context.Background(), "H"+ipkey[0], "root")
		BalanceCon.Client.Put(context.Background(), "lock", "true")

		//add load balance in nginx when peer create
		LoadBalance(configs.Confile.MinerData.ServiceIpAddr+":"+configs.Confile.MinerData.ServicePort, "")

		go GetAliveKey()
		//go stateMachine()
	case "root":
		fmt.Println("this is root node")
		if role.PeerInfo.AllowJoin {
			exec_shell_immediately(commandetcd)
			err := OpenAuthBalance()
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("Open Etcd Balance With Auth Error: %v", err)
				checkErr(err)
			}
			ipkey := strings.Split(configs.Confile.RotationModule.Endpoint, ":")
			BalanceCon.Client.Put(context.Background(), "/"+ipkey[0], "root")
			BalanceCon.Client.Put(context.Background(), "H"+ipkey[0], "root")
		} else {
			if role.Err != "" {
				logger.InfoLogger.Sugar().Infof(role.Err)
				checkErr(errors.New(role.Err))
			}
		}
		//add load balance in nginx when peer create
		LoadBalance(configs.Confile.MinerData.ServiceIpAddr+":"+configs.Confile.MinerData.ServicePort, "")
		go GetAliveKey()
		go stateMachine()
	case "normal":
		fmt.Println("this is normal node")
		go LoopJoinCluster()
		if role.PeerInfo.AllowJoin {
			exec_shell_immediately(commandetcd)
			err := OpenAuthBalance()
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("normal role open auth balance err:%s", err.Error())
			}
			fmt.Println("normal role open auth balance")
		} else {
			if role.Err != "" {
				logger.InfoLogger.Sugar().Infof(role.Err)
			}
		}

	}
}

func EtcdStartCommand() (commandetcd string, role PeerInfoResult) {
	resp, err := ce.Ci.GetDataOnChain()
	if err != nil || len(resp.Ip) == 0 {
		logger.ErrLogger.Sugar().Errorf("GetDataOnChain Error:%v", err.Error())
		checkErr(err)
		os.Exit(0)
	}
	configs.Cd0url = string(resp.Ip)
	fmt.Println("Genesis Address Is:", string(resp.Ip))
	role = RequestStatus(
		configs.Cd0url,
		configs.Confile.RotationModule.Username,
		configs.Confile.RotationModule.Password,
		configs.Confile.RotationModule.InitialAdvertisePeerUrls,
		configs.Confile.RotationModule.Name)
	exsistpeer := ""
	if role.Role == "genesis" {
		exsistpeer = configs.Confile.RotationModule.Name + "=" + configs.Confile.RotationModule.InitialAdvertisePeerUrls
		configs.InitialClusterState = "new"
		err := ce.Tk.RegisterEtcdOnChain(configs.Confile.RotationModule.InitialClusterToken)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Register etcd token fail:%v", err.Error())
			checkErr(err)
		}
	} else {
		exsistpeer = configs.Confile.RotationModule.Name + "=" + configs.Confile.RotationModule.InitialAdvertisePeerUrls
		for name, ip := range role.Participants {
			exsistpeer = exsistpeer + "," + name + "=" + ip
		}
		configs.InitialClusterState = "existing"
	}
	respt, err := ce.Tk.GetDataOnChain()
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("Get etcd token on chain fail error:%v", err.Error())
		checkErr(err)
	}
	InitialClusterToken := ""
	if len(resp.Ip) != 0 {
		InitialClusterToken = string(respt.Ip)
	} else {
		logger.ErrLogger.Sugar().Errorf("Can't get the token of etcd cluster pelase try again or check your network")
		checkErr(errors.New("Can't get the token of etcd cluster pelase try again or check your network"))
	}

	//commandetcd = "nohup etcd --name " + configs.Confile.RotationModule.Name +
	//	" --listen-client-urls " + configs.Confile.RotationModule.ListenClientUrls +
	//	" --advertise-client-urls " + configs.Confile.RotationModule.AdvertiseClientUrls +
	//	" --listen-peer-urls " + configs.Confile.RotationModule.ListenPeerUrls +
	//	" --initial-advertise-peer-urls " + configs.Confile.RotationModule.InitialAdvertisePeerUrls +
	//	" --initial-cluster-state " + configs.InitialClusterState +
	//	" --initial-cluster " + exsistpeer +
	//	" --initial-cluster-token " + InitialClusterToken
	//othercom := " >./log_etcd.log 2>&1 &"
	//commandetcd += othercom

	commandetcd = " docker run -itd -p 2380:2380 -p 2379:2379 --name cess_etcd --network host localhost/cess_etcd /usr/local/bin/etcd" +
		" -name " + configs.Confile.RotationModule.Name +
		" -advertise-client-urls " + configs.Confile.RotationModule.AdvertiseClientUrls +
		" -listen-client-urls " + configs.Confile.RotationModule.ListenClientUrls +
		" -initial-advertise-peer-urls " + configs.Confile.RotationModule.InitialAdvertisePeerUrls +
		" -listen-peer-urls " + configs.Confile.RotationModule.ListenPeerUrls +
		" -initial-cluster-token " + InitialClusterToken +
		" -initial-cluster " + exsistpeer +
		" -initial-cluster-state " + configs.InitialClusterState

	return
}

func GetAliveKey() {
	for range time.Tick(time.Second * 5) {
		res, err := BalanceCon.Client.Get(context.Background(), "/", clientv3.WithPrefix(), func(op *clientv3.Op) {
		})
		if err != nil {
			logger.ErrLogger.Sugar().Infof("Get Cluster key prefix / error: %v", err.Error())
		} else {
			logger.WatcherLogger.Sugar().Infof("Watcher queue length: %v", len(res.Kvs))
			for _, v := range res.Kvs {
				//暂时不关注root，浪费携程
				logger.WatcherLogger.Sugar().Infof("watching:%v", string(v.Key))
				if string(v.Value) == "root" {
					continue
				}
				_, ok := WatcherMap.Load(string(v.Key))
				if ok {
					continue
				} else {
					logger.WatcherLogger.Sugar().Infof("add a normal user watcher:%v", string(v.Key))
					fmt.Println("add a normal user watcher:", string(v.Key))
					WatcherMap.Store(string(v.Key), time.Now().Format("2006-01-02 15:04:05"))
					ClusterNum.Add(1)
					go WatchClusterNum(string(v.Key))
				}
			}
		}
	}
}

func WatchClusterNum(key string) {
	ctx, cancel := context.WithCancel(context.Background())
	res := BalanceCon.Client.Watcher.Watch(ctx, key)
	logger.WatcherLogger.Sugar().Infof("Watch the key :%v", key)
loop:
	for wresp := range res {
		for _, ev := range wresp.Events {
			if ev.Type == 1 {
				ClusterNum.Sub(1)
				logger.WatcherLogger.Sugar().Infof("%s is exit from cluster,normal peers number is:%d", ev.Kv.Key, ClusterNum.Load())
				DeleteMember(string(ev.Kv.Key))
				WatcherMap.Delete(key)
				cancel()
				break loop
			}
		}
	}
}

func DeleteMember(key string) {
	var memberId uint64
	resp, err := BalanceCon.Client.MemberList(context.Background())
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("MemberList error when need deletemember")
	}
	for _, Member := range resp.Members {
		if len(Member.PeerURLs) == 0 {
			continue
		}
		if strings.Contains(Member.PeerURLs[0], key) {
			memberId = Member.ID
		}
	}
	logger.WatcherLogger.Sugar().Infof("Time over DeleteMember ,memberId is:%v", memberId)
	_, err = BalanceCon.Client.MemberRemove(context.Background(), memberId)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("MemberRemove error when need deletemember:%s", err.Error())
	}
}

func RentKey(key string) {
	BalanceRent, err := AuthBalancer(configs.Confile.RotationModule.Endpoint)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("Rent Key BalanceRent happened issue: %v", err.Error())
		checkErr(err)
	}
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	resp, err := BalanceRent.Client.Grant(ctx, int64(BalanceCon.TTL))
	cancel()
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("key %q grant with TTL fail: %s", key, err.Error())
		return
	}
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	if _, err = BalanceRent.Client.Put(ctx, "/"+key, time.Now().Format("2006-01-02 15:04:05"), clientv3.WithLease(resp.ID)); err != nil {
		logger.ErrLogger.Sugar().Errorf("put key %q grant fail: %s", key, err.Error())
	}
	checkErr(err)
	cancel()
	err = BalanceRent.Client.Close()
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("Close etcd pool err: %s", err.Error())
	}
}

func LoopJoinCluster() {
	fmt.Println("start looping")
	for range time.Tick(time.Second * 5) {
		fmt.Println("-----loop1-----")
		if !CommandPid() {
			logger.InfoLogger.Sugar().Infof("Normal Loop Join Cluster...")
			commandetcd, role := EtcdStartCommand()
			fmt.Println("-----loop2-----", role)
			if role.Err != "" {
				logger.InfoLogger.Sugar().Infof(configs.Confile.RotationModule.Username+"("+role.Role+") join cluster info:", role.Err)
				continue
			}
			if role.AllowJoin {
				CleanContainer()
				//tools.CleanLocalRecord(".etcd")
				logger.InfoLogger.Sugar().Infof(configs.Confile.RotationModule.Username + "(" + role.Role + ")" + " info:Join in this cluster be allowed for now")
				exec_shell_immediately(commandetcd)
				fmt.Println(commandetcd)
				logger.InfoLogger.Sugar().Infof(configs.Confile.RotationModule.Username + " join the cluster successful!")

				err := OpenAuthBalance()
				if err != nil {
					logger.ErrLogger.Sugar().Errorf("LoopJoinCluster OpenAuthBalance ERR:", err)
				} else {
					fmt.Println("Open auth balance successful!")
				}
				continue
			} else {
				if role.Err != "" {
					logger.InfoLogger.Sugar().Infof(role.Err)
				}
			}
		}
	}
}
