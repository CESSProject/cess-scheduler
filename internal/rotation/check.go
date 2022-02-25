package rotation

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"scheduler-mining/configs"
	"scheduler-mining/internal/logger"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func exec_shell(s string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", s)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	return out.String(), err
}

func exec_shell_immediately(s string) {
	cmd := exec.Command("/bin/bash", "-c", s)
	err := cmd.Start()
	if err != nil {
		logger.ErrLogger.Sugar().Errorf(err.Error())
	}
	time.Sleep(time.Second)
	return
}

func CommandPid() (ok bool) {
	commandpid := "curl 127.0.0.1:2379"
	_, err := exec_shell(commandpid)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 7") {
			return false
		} else {
			return true
		}
	}
	return true
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func DetectJoinInfo(ctx *gin.Context) {
	var query JoinInfo
	var result PeerInfo
	if err := ctx.ShouldBindBodyWith(&query, binding.JSON); err != nil {
		ReplyFail(ctx, "request para error!")
		return
	}
	//root节点加入root组，普通节点加入普通权限角色
	if query.Password == configs.Confile.RotationModule.Password && query.Username == "root" {
		localIP := "curl ifconfig.me"
		res, err := exec_shell(localIP)
		if err != nil {
			panic("please check your Internet")
		}
		cdurl := strings.Split(configs.Cd0url, ":")
		queryUrlh := strings.Split(query.LocalUrl, ":")
		queryUrl := strings.Trim(queryUrlh[1], "//")
		if cdurl[0] == res && queryUrl == res {
			result.Role = "genesis"
			result.AllowJoin = true
			result.Participants = make(map[string]string, 0)
			ReplySuccess(ctx, result)
			return
		} else {
			var rootnum int
			list, err := BalanceCon.Client.MemberList(ctx)
			if err != nil {
				result.Err = "err:" + err.Error()
				ReplyFail(ctx, result)
				return
			}
			result.Participants = make(map[string]string, len(list.Members))
			memberidMap := make(map[string]uint64, len(list.Members))
			for _, peer := range list.Members {
				if len(peer.PeerURLs) == 0 {
					continue
				}
				result.Participants[peer.Name] = peer.PeerURLs[0]
				memberidMap[peer.PeerURLs[0]] = peer.ID
			}
			roots, _ := BalanceCon.Client.Get(context.Background(), "/", clientv3.WithPrefix())
			for _, v := range roots.Kvs {
				if string(v.Value) == "root" {
					peerip := strings.Trim(string(v.Key), "/")
					curlip := "http://" + peerip + ":2380"
					_, err := exec_shell("curl " + curlip)
					if err != nil {
						if strings.Contains(err.Error(), "exit status 7") {
							for name, ip := range result.Participants {
								if ip == curlip {
									delete(result.Participants, name)
								}
							}
							BalanceCon.Client.MemberRemove(context.Background(), memberidMap[curlip])
							continue
						}
					}
					rootnum++
				}
			}
			result.Role = "root"
			if rootnum < configs.Confile.RotationModule.RootTotal {
				result.AllowJoin = true
				peerURLs := []string{query.LocalUrl}
				//规定以2380作为etcd广播口
				if queryUrlh[2] == "2380" {
					_, err := BalanceCon.Client.MemberAdd(ctx, peerURLs)
					if err != nil {
						logger.ErrLogger.Sugar().Errorf("MemberAdd error:%v", err.Error())
						result.AllowJoin = false
						result.Err = "err:" + err.Error()
						ReplyFail(ctx, result)
						return
					}
				}
			} else {
				result.AllowJoin = false
				result.Err = "Cluster root has reached the upper limit"
				ReplyFail(ctx, result)
			}
			ReplySuccess(ctx, result)
			return
		}
	} else {
		if query.Username == "root" {
			result.AllowJoin = false
			result.Err = "Root password is error ,please check your password!"
			ReplyFail(ctx, result)
			return
		} else {
			result.Role = "normal"
			list, err := BalanceCon.Client.MemberList(ctx)
			if err != nil {
				result.Err = "err:" + err.Error()
				ReplyFail(ctx, result)
				return
			}
			result.Participants = make(map[string]string, len(list.Members))
			for _, peer := range list.Members {
				if len(peer.PeerURLs) == 0 {
					continue
				}
				result.Participants[peer.Name] = peer.PeerURLs[0]
				if peer.Name == query.Name {
					result.AllowJoin = false
					result.Err = "This node name already exists in the cluster. Please change the name"
					ReplyFail(ctx, result)
					return
				}
			}
			resp, code := RequestJoinCluster([]string{query.LocalUrl}, query.Username, query.Password, result.Role)
			if code == http.StatusOK {
				result.AllowJoin = resp.Join
				result.Err = resp.Err
				ReplySuccess(ctx, result)
				return
			} else {
				result.AllowJoin = resp.Join
				result.Err = resp.Err
				ReplyFail(ctx, result)
				return
			}
		}
	}
	return
}

func TestPolling(ctx *gin.Context) {
	fmt.Println("TestPolling ......")
	return
}

func CheckClusterSpaceEnough(ctx *gin.Context) {
	var query RequestJoin
	var result JoinResult

	if err := ctx.ShouldBindBodyWith(&query, binding.JSON); err != nil {
		result.Err = "request para error!"
		ReplyFail(ctx, result)
		return
	}
	//Prevent external nodes from joining, and quickly restart the service when the center does not delete the understand node
	memberlist, err := BalanceCon.Client.MemberList(context.Background())
	if err != nil {
		result.Join = false
		result.Err = err.Error()
		ReplyFail(ctx, result)
		return
	}
	for _, mem := range memberlist.Members {
		for _, v := range mem.PeerURLs {
			if len(mem.PeerURLs) == 0 {
				continue
			}
			if v == query.PeerURLs[0] {
				result.Join = false
				result.Err = "This peer is understanding in cluster , please wait for 5 seconds."
				ReplySuccess(ctx, result)
				return
			}
		}
	}
	queryUrlh := strings.Split(query.PeerURLs[0], ":")
	queryUrl := strings.Trim(queryUrlh[1], "//")

	ctx1, cancel := context.WithTimeout(context.Background(), time.Second)
	res, err := BalanceCon.Client.Get(ctx1, "H"+queryUrl)
	cancel()
	defer BalanceCon.Client.Put(context.Background(), "lock", "true")

	//第一次进入
	if len(res.Kvs) == 0 {
		fmt.Println("[firstEnter]The number of normal peer in this cluster is:", ClusterNum.Load())
		if configs.Confile.RotationModule.NormalTotal > ClusterNum.Load() {
			lock, err := BalanceCon.Client.Get(context.Background(), "lock")
			if err != nil {
				result.Err = "the very importance param 'lock' is disappear! Cluster have big problem"
				ReplyFail(ctx, result)
				panic("the very importance param 'lock' is disappear!")
				return
			}
			for _, locv := range lock.Kvs {
				if string(locv.Value) == "false" {
					result.Join = false
					result.Err = "The system is locked, please try again later"
					ReplySuccess(ctx, result)
					return
				} else {
					_, err = BalanceCon.Client.Put(context.Background(), "lock", "false")
					if err != nil {
						result.Join = false
						result.Err = "lock fail when join the cluster"
						ReplyFail(ctx, result)
						return
					}
				}
			}
			createAcc := true
			//userinfo, err := BalanceCon.Client.UserGet(context.Background(), query.Username)
			userinfo, err := BalanceCon.Client.UserList(context.Background())
			for _, user := range userinfo.Users {
				if user == query.Username {
					fmt.Println("already exist this account:", query.Username)
					createAcc = false
					cli, err := clientv3.New(clientv3.Config{
						Endpoints: strings.Split("127.0.0.1:2379", ","),
						Username:  query.Username,
						Password:  query.Password,
					})
					if err != nil {
						logger.InfoLogger.Sugar().Infof("already exists this username:%v and wrong password:%v", query.Username, query.Password)
						result.Join = false
						result.Err = fmt.Sprintf("already exists this username:%v and wrong password:%v", query.Username, query.Password)
						ReplyFail(ctx, result)
					} else {
						cli.Close()
					}
				}
			}

			if createAcc {
				fmt.Printf("first register this account:%s , password:%s", query.Username, query.Password)
				//第一次进集群需要给予其注册账号并赋普通权限,有账号之后才能注册后面内容
				_, err = BalanceCon.Client.UserAdd(context.Background(), query.Username, query.Password)
				logger.InfoLogger.Sugar().Infof("first add this normal Username:%v Password:%v", query.Username, query.Password)
				_, err = BalanceCon.Client.UserGrantRole(context.Background(), query.Username, query.Role)
				if err != nil {
					result.Join = false
					result.Err = err.Error()
					ReplyFail(ctx, result)
					return
				}
			}

			//更新加入集群的历史记录
			ctx1, cancel = context.WithTimeout(context.Background(), time.Second)
			BalanceCon.Client.Put(ctx1, "H"+queryUrl, strconv.FormatInt(time.Now().Unix(), 10))
			cancel()
			RentKey(queryUrl)
			ctx1, cancel = context.WithTimeout(context.Background(), time.Second)
			_, err = BalanceCon.Client.Cluster.MemberAdd(ctx1, query.PeerURLs)
			cancel()
			if err != nil {
				result.Join = false
				result.Err = err.Error()
				ReplyFail(ctx, result)
				return
			}
			result.Join = true
			ReplySuccess(ctx, result)
			return
		} else {
			result.Join = false
			result.Err = fmt.Sprintf("[firstEnter]The number of nodes has reached the upper limit, please wait, please wait,normal peer in this cluster is:%d", ClusterNum.Load())
			ReplySuccess(ctx, result)
			return
		}
	} else {
		for _, v := range res.Kvs {
			pasttime, _ := strconv.Atoi(string(v.Value))
			if time.Now().Unix()-int64(pasttime) > configs.Confile.RotationModule.PollingTime {
				fmt.Println("[againEnter]The number of normal peer in this cluster is:", ClusterNum.Load())
				if configs.Confile.RotationModule.NormalTotal > ClusterNum.Load() {
					//allow join this cluster
					ctx1, cancel = context.WithTimeout(context.Background(), time.Second)
					_, err := BalanceCon.Client.Cluster.MemberAdd(ctx1, query.PeerURLs)
					cancel()
					//update this url history
					ctx1, cancel = context.WithTimeout(context.Background(), time.Second)
					BalanceCon.Client.Put(ctx1, "H"+queryUrl, strconv.FormatInt(time.Now().Unix(), 10))
					cancel()
					if err != nil {
						result.Join = false
						result.Err = err.Error()
						ReplyFail(ctx, result)
						return
					}
					RentKey(queryUrl)
					result.Join = true
					ReplySuccess(ctx, result)
					return
				} else {
					result.Join = false
					result.Err = fmt.Sprintf("[againEnterThe number of nodes has reached the upper limit, please wait,normal peer in this cluster is:%d", ClusterNum.Load())
					ReplySuccess(ctx, result)
					return
				}
			} else {
				result.Join = false
				result.Err = "Polling time is not up yet, please wait again!"
				ReplySuccess(ctx, result)
				return
			}
		}
	}
	result.Err = "Unknown Error Happen"
	ReplyFail(ctx, result)
	return
}

func CleanContainer() {
	command := "docker stop cess_etcd &&docker rm cess_etcd"
	exec_shell_immediately(command)
}
