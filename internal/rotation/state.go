package rotation

import (
	"context"
	"scheduler-mining/configs"
	ce "scheduler-mining/internal/chain"
	"scheduler-mining/internal/logger"
	"strings"
	"time"
)

func stateMachine() {
	for range time.Tick(time.Second) {
		curlip := "http://" + configs.Cd0url
		_, err := exec_shell("curl " + curlip)

		ipnp := strings.Split(configs.Cd0url, ":")
		if err != nil {
			if err.Error() == "exit status 7" && configs.Cd0url != configs.Confile.RotationModule.Endpoint {
				logger.InfoLogger.Sugar().Infof("The Genesis Is Died ")
				list, err := BalanceCon.Client.MemberList(context.Background())
				if err != nil {
					logger.ErrLogger.Sugar().Errorf("get member list fail,cluster died")
				}
				for _, peer := range list.Members {
					if strings.Contains(peer.PeerURLs[0], ipnp[0]) {
						_, err = BalanceCon.Client.MemberRemove(context.Background(), peer.ID)
						if err != nil {
							logger.ErrLogger.Sugar().Errorf("remove died genesis :%v fail because :%v", peer.Name, err.Error())
						}
						_, err = BalanceCon.Client.Delete(context.Background(), "/"+ipnp[0])
						if err != nil {
							logger.ErrLogger.Sugar().Errorf("Delete Key With prefix / When /%v Died , But Fail: %v", ipnp[0], err.Error())
						}
					}
				}
				err = ce.Ci.RegisterEtcdOnChain(configs.Confile.RotationModule.Endpoint)
				if err != nil {
					logger.ErrLogger.Sugar().Errorf("Fail Register My Etcd Address err:%v", err.Error())
				} else {
					configs.Cd0url = configs.Confile.RotationModule.Endpoint
				}
			}
		}
	}
}
