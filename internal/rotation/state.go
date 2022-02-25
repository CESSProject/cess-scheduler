package rotation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"scheduler-mining/configs"
	ce "scheduler-mining/internal/chain"
	"scheduler-mining/internal/logger"
	"strings"
	"time"
)

type Polling struct {
	Ip    string `json:"ip"`
	Order string `json:"order"`
}

func stateMachine() {
	fmt.Println("[stateMachine]:start etcd stateMachine")
	var outnum = 0
	for range time.Tick(time.Second) {
		list, err := BalanceCon.Client.MemberList(context.Background())
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("get member list fail,cluster died")
		}
		for _, peer := range list.Members {
			if len(peer.ClientURLs) == 0 {
				continue
			}
			dkslice := strings.Split(peer.ClientURLs[0], ":")
			dk := strings.Trim(dkslice[1], "//")
			_, err = exec_shell("curl " + peer.ClientURLs[0])
			//_, err1 := exec_shell("curl " + dk + ":" + configs.Confile.MinerData.ServicePort)
			if err != nil {
				if strings.Contains(err.Error(), "exit status 7") {
					_, err = BalanceCon.Client.MemberRemove(context.Background(), peer.ID)
					if err != nil {
						logger.ErrLogger.Sugar().Errorf("remove died peer :%s fail because :%s", peer.Name, err.Error())
					}

					_, err = BalanceCon.Client.Delete(context.Background(), "/"+dk)
					if err != nil {
						logger.ErrLogger.Sugar().Errorf("Delete Key with prefix / When /%s Died , But Fail: %s", dk, err.Error())
					}
					//delete load balance in nginx
					LoadBalance(dk, "Delete")
					logger.InfoLogger.Sugar().Infof("[stateMachine]:Delete the key:%s from polling", dk)
					//if myIp is not genesis's ip
					if strings.Contains(peer.ClientURLs[0], configs.Cd0url) && configs.Cd0url != configs.Confile.RotationModule.Endpoint {
						err = ce.Ci.RegisterEtcdOnChain(configs.Confile.RotationModule.Endpoint)
						if err != nil {
							logger.ErrLogger.Sugar().Errorf("Fail Register My Etcd Address err:%v", err.Error())
						} else {
							resp, err := ce.Ci.GetDataOnChain()
							if err != nil || len(resp.Ip) == 0 {
								logger.ErrLogger.Sugar().Errorf("[stateMachine]:get genesis address after register error:%s", err)
							}
							logger.InfoLogger.Sugar().Infof("[stateMachine]:update genesis ip successful!")
							configs.Cd0url = string(resp.Ip)
						}
					}
				}
			}
		}
		outnum++
		if outnum%7 == 0 {
			resp, err := ce.Ci.GetDataOnChain()
			if err != nil || len(resp.Ip) == 0 {
				logger.ErrLogger.Sugar().Errorf("[stateMachine]:get genesis address per 7 seconds fail:%s", err)
			}
			logger.InfoLogger.Sugar().Infof("[stateMachine]:update genesis ip per 7 seconds successful!")
			configs.Cd0url = string(resp.Ip)
			logger.InfoLogger.Sugar().Infof("[stateMachine]The genesis is:", configs.Cd0url, "for now")
		}
		//
		//curlip := "http://" + configs.Cd0url
		//_, err := exec_shell("curl " + curlip)
		//
		//ipnp := strings.Split(configs.Cd0url, ":")
		//if err != nil {
		//	if strings.Contains(err.Error(), "exit status 7") && configs.Cd0url != configs.Confile.RotationModule.Endpoint {
		//		logger.InfoLogger.Sugar().Infof("The Genesis Is Died ")
		//		list, err := BalanceCon.Client.MemberList(context.Background())
		//		if err != nil {
		//			logger.ErrLogger.Sugar().Errorf("get member list fail,cluster died")
		//		}
		//		for _, peer := range list.Members {
		//			if strings.Contains(peer.PeerURLs[0], ipnp[0]) {
		//				_, err = BalanceCon.Client.MemberRemove(context.Background(), peer.ID)
		//				if err != nil {
		//					logger.ErrLogger.Sugar().Errorf("remove died genesis :%v fail because :%v", peer.Name, err.Error())
		//				}
		//				_, err = BalanceCon.Client.Delete(context.Background(), "/"+ipnp[0])
		//				if err != nil {
		//					logger.ErrLogger.Sugar().Errorf("Delete Key With prefix / When /%v Died , But Fail: %v", ipnp[0], err.Error())
		//				}
		//			}
		//		}
		//		err = ce.Ci.RegisterEtcdOnChain(configs.Confile.RotationModule.Endpoint)
		//		if err != nil {
		//			logger.ErrLogger.Sugar().Errorf("Fail Register My Etcd Address err:%v", err.Error())
		//		} else {
		//			resp, err := ce.Ci.GetDataOnChain()
		//			if err != nil || len(resp.Ip) == 0 {
		//				logger.ErrLogger.Sugar().Errorf("[stateMachine]:get genesis address after register error:%s", err)
		//			}
		//			configs.Cd0url = string(resp.Ip)
		//		}
		//	}
		//}
		//logger.InfoLogger.Sugar().Infof("[stateMachine]The genesis is:", configs.Cd0url, "for now")

	}
}

func LoadBalance(Ip, Order string) {
	var para = Polling{
		Ip:    Ip,
		Order: Order,
	}
	bodydata_Marshal, err := json.Marshal(para)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("LoadBalance fail when marshal para:%s", err.Error())
		return
	}

	url := "http://139.224.19.104:9700/editconf"
	var resp = new(http.Response)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodydata_Marshal))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("Post Polling fail:%s", err.Error())
		return
	}
	defer resp.Body.Close()
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			logger.ErrLogger.Sugar().Errorf("Post to edit nginx fail ip:%s,order:%s", Ip, Order)
			return
		}
	}

	return
}
