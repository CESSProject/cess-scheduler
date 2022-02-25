package rotation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"scheduler-mining/configs"
	"scheduler-mining/internal/logger"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/gin-gonic/gin"
	"go.uber.org/atomic"
)

type Balancer struct {
	TTL         int
	Interval    time.Duration
	Client      *clientv3.Client
	serviceList []*ServiceInfo
}

type ServiceInfo struct {
	IP     string
	Port   int
	Worker int
	Load   int
}

type JoinInfo struct {
	Password string `json:"Password"`
	Username string `json:"Username"`
	LocalUrl string `json:"LocalUrl"`
	Name     string `json:"Name"`
}
type RequestJoin struct {
	PeerURLs []string `json:"PeerURLs"`
	Username string   `json:"Username"`
	Password string   `json:"Password"`
	Role     string   `json:"Role"`
}

type PeerInfoResult struct {
	PeerInfo `json:"Result"`
}

type PeerInfo struct {
	Role         string            `json:"Role"`
	AllowJoin    bool              `json:"AllowJoin"`
	Participants map[string]string `json:"Participants"` //name:IP，当前存在集群内的节点有哪些
	Err          string            `json:"Err"`
}

type RequestJoinResult struct {
	JoinResult `json:"Result"`
}
type JoinResult struct {
	Join bool   `json:"Join"`
	Err  string `json:"Err"`
}

var BalanceCon *Balancer
var ClusterNum atomic.Int64
var WatcherMap sync.Map

func OpenBalance() error {
	var err error
	if err != nil {
		return err
	}
	BalanceCon, err = NewBalancer(configs.Confile.RotationModule.Endpoint)
	if err != nil {
		return err
	}
	return err
}

func OpenAuthBalance() error {
	var err error
	if err != nil {
		return err
	}
	BalanceCon, err = AuthBalancer(configs.Confile.RotationModule.Endpoint)
	if err != nil {
		return err
	}
	return err
}

func NewBalancer(endpoints string) (*Balancer, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(endpoints, ","),
		DialTimeout: time.Second * 2,
	})

	if err != nil {
		return nil, err
	}

	BalancerN := &Balancer{
		Client: cli,
	}

	return BalancerN, err
}

func AuthBalancer(endpoints string) (*Balancer, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: strings.Split(endpoints, ","),
		Username:  configs.Confile.RotationModule.Username,
		Password:  configs.Confile.RotationModule.Password,
	})

	if err != nil {
		return nil, err
	}

	BalancerA := &Balancer{
		Client:   cli,
		TTL:      60, //租约时间12秒
		Interval: 10, //每10秒上报一次
	}

	return BalancerA, err
}

func ReplySuccess(ctx *gin.Context, r interface{}) {
	ctx.JSON(200, gin.H{
		"Result": r,
		"Status": "Success",
	})
}

func ReplyFail(ctx *gin.Context, r interface{}) {
	ctx.JSON(400, gin.H{
		"Result": r,
		"Status": "Fail",
	})
}

func RequestStatus(cd0url, Username, Password, localUrl, name string) (res PeerInfoResult) {
	fmt.Println(cd0url, Username, Password, localUrl, name, "===========")
	if len(cd0url) == 0 || len(Username) == 0 || len(Password) == 0 || len(localUrl) == 0 || len(name) == 0 {
		fmt.Println(cd0url, Username, Password, localUrl, name)
		logger.ErrLogger.Sugar().Errorf("Bad Parameter!Please Check your conf.yaml")
		checkErr(errors.New("Bad Parameter!Please Check your conf.yaml"))
	}
	var para = JoinInfo{Password: Password, Username: Username, LocalUrl: localUrl, Name: name}
	bodydata_Marshal, err := json.Marshal(para)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("Marshal RequestStatus fail:", err)
		checkErr(err)
	}
	urltmp := strings.Split(cd0url, ":")
	url := "http://" + urltmp[0] + ":" + configs.Confile.MinerData.ServicePort + "/detectjoininfo"
	var resp = new(http.Response)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodydata_Marshal))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		checkErr(err)
		return
	}
	if resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}
			json.Unmarshal(body, &res)
			return
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			checkErr(err)
			json.Unmarshal(body, &res)
			if res.Err != "" {
				logger.ErrLogger.Sugar().Errorf("Status Detect Fail:%v", res.Err)
			}
			checkErr(errors.New(res.Err))
		}
	}
	return
}

func RequestJoinCluster(peerURLs []string, Username, Password, Role string) (res RequestJoinResult, StatusCode int) {
	var para = RequestJoin{PeerURLs: peerURLs, Username: Username, Password: Password, Role: Role}
	bodydata_Marshal, err := json.Marshal(para)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("Marshal RequestJoinCluster fail:%v", err)
		res.Join = false
		res.Err = "Marshal RequestJoinCluster fail:" + err.Error()
		return
	}
	urltmp := strings.Split(configs.Cd0url, ":")
	url := "http://" + urltmp[0] + ":" + configs.Confile.MinerData.ServicePort + "/checkclusterspaceenough"
	var resp = new(http.Response)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodydata_Marshal))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		checkErr(err)
		return
	}
	StatusCode = resp.StatusCode
	if resp != nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		err = json.Unmarshal(body, &res)
		fmt.Println(body)
		if err != nil {
			checkErr(err)
		}
		return
	}
	return
}
