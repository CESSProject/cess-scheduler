package task

func Run() {
	var (
		channel_1 = make(chan bool, 1)
		channel_2 = make(chan bool, 1)
		channel_3 = make(chan bool, 1)
		channel_4 = make(chan bool, 1)
	)
	go task_SyncMinersInfo(channel_1)
	go task_ValidateProof(channel_2)
	go task_SubmitFillerMeta(channel_3)
	go task_GenerateFiller(channel_4)
	for {
		select {
		case <-channel_1:
			go task_SyncMinersInfo(channel_1)
		case <-channel_2:
			go task_ValidateProof(channel_2)
		case <-channel_3:
			go task_SubmitFillerMeta(channel_3)
		case <-channel_4:
			go task_GenerateFiller(channel_4)
		}
	}
}
