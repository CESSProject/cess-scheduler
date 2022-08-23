package task

func Run() {
	var (
		channel_1 = make(chan bool, 1)
		//channel_2 = make(chan bool, 1)
		channel_3 = make(chan bool, 1)
		channel_4 = make(chan bool, 1)
		channel_5 = make(chan bool, 1)
		channel_6 = make(chan bool, 1)
	)
	go task_SyncMinersInfo(channel_1)
	//go task_RecoveryFiles(channel_2)
	go task_ValidateProof(channel_3)
	go task_ClearAuthMap(channel_4)
	go task_SubmitFillerMeta(channel_5)
	go task_GenerateFiller(channel_6)
	for {
		select {
		case <-channel_1:
			go task_SyncMinersInfo(channel_1)
		// case <-channel_2:
		// 	go task_RecoveryFiles(channel_2)
		case <-channel_3:
			go task_ValidateProof(channel_3)
		case <-channel_4:
			go task_ClearAuthMap(channel_4)
		case <-channel_5:
			go task_SubmitFillerMeta(channel_5)
		case <-channel_6:
			go task_GenerateFiller(channel_6)
		}
	}
}
