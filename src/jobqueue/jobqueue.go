package jobqueue

import (
	"log"
	"sync"
	"time"
)

const (
	QUEUE_MAX = 1000
)

// 佇列種類
const (
	QUEUE_KIND_WALLET_DEPOSIT    int = iota // 0:錢包提款
	QUEUE_KIND_WALLET_WITHDARW              // 1:錢包存款
	QUEUE_KIND_WALLET_AMOUNT_GET            // 2:取得用戶錢包數量
)

// 佇列待處理工作項目物件
type QueueWorkInfo struct {
	JobId         int64       //佇列Id
	PlatformID    int         //第三方平台編號
	UserId        int64       //用戶ID
	TransactionId string      //交易ID
	QueueKind     int         //參考 通道種類 QUEUE_KIND_WALLET_DEPOSIT
	DoSomeThing   interface{} //要作的事情

	WaitQueueInfo *WaitQueueInfo
}

type WaitQueueInfo struct {
	JobId        int64                  `json:"jobId"`
	IsCloseQueue bool                   `json:"is_closeQueue"` // 是否關閉通道了
	WaitQueue    chan WaitQueueInfo     `json:"-"`
	IsGet        bool                   `json:"is_get"` // 是否有回傳資料
	Data         map[string]interface{} `json:"data"`   // 回傳的資料
}

type QueueWorkManager struct {
	jobQueue chan QueueWorkInfo // 工作佇列
	jobCount int64              // 工作柱列id
	mutex    sync.Mutex         // 關閉資源
}

// 取得實例
var jobManager *QueueWorkManager

func InstanceGet() *QueueWorkManager {
	if jobManager == nil {

		jobManager = &QueueWorkManager{

			jobQueue: make(chan QueueWorkInfo, QUEUE_MAX),
		}

	}
	return jobManager
}

// 每個使用者配發一個專屬通道
func NewJobQueue() *QueueWorkManager {
	jobManager := &QueueWorkManager{
		jobQueue: make(chan QueueWorkInfo, QUEUE_MAX),
	}

	log.Printf("NewJobQueue addr:%p, addr2:%p", jobManager, jobManager.jobQueue)
	return jobManager
}

// 工作佇列塞入
func (manager *QueueWorkManager) Insert(platformID int, userId int64, transactionId string, queueKind int, needWait bool, doSomeThing interface{}) (waitinfo *WaitQueueInfo) {

	// 佇列流水號+1
	manager.mutex.Lock()
	manager.jobCount++
	manager.mutex.Unlock()

	if needWait {
		waitinfo = &WaitQueueInfo{
			JobId:     manager.jobCount,
			WaitQueue: make(chan WaitQueueInfo),
		}
	}

	// 組合jobqueue資訊
	queueWork := QueueWorkInfo{

		JobId:         manager.jobCount,
		PlatformID:    platformID,
		UserId:        userId,
		TransactionId: transactionId,
		QueueKind:     queueKind,
		DoSomeThing:   doSomeThing,

		WaitQueueInfo: waitinfo,
	}

	// 寫入job佇列通道中
	manager.jobQueue <- queueWork

	// 統計通道使用率
	cnt := len(manager.jobQueue)
	size := cap(manager.jobQueue)
	log.Printf("通道使用: [%v/%v]", cnt, size)

	// 回傳等待接收通道
	return
}

// 立即等待佇列回應
func (manager *QueueWorkManager) WaitQueue(waitinfo *WaitQueueInfo) (data WaitQueueInfo) {

	if waitinfo == nil {
		return
	}

	defer func() {
		log.Printf("關閉通道 jobId:%v", waitinfo.JobId)
		manager.mutex.Lock()
		defer manager.mutex.Unlock()
		close(waitinfo.WaitQueue)

		if waitinfo != nil {
			waitinfo.IsCloseQueue = true
		}
	}()

	log.Printf("WaitQueue 開始等待 jobId:%v", waitinfo.JobId)
	//timeout := time.After(time.Second * 10)
	select {
	case data = <-waitinfo.WaitQueue:
		log.Printf("收到jobqueue data:%+v", data)

		break

		// case <-timeout:
		// 	data.IsGet = false //timeout
		// 	log.Printf("WaitQueue timeout 離開 select, jobId:%v", waitinfo.JobId)
		// 	break
	}

	log.Printf("WaitQueue 離開 jobId:%v", waitinfo.JobId)
	return
}

func (manager *QueueWorkManager) Run() {

	go func() {

		log.Printf("job通道監聽-開始")

		for job := range manager.jobQueue {

			log.Printf("job:%v", job)
			switch job.QueueKind {
			case QUEUE_KIND_WALLET_DEPOSIT:
				log.Printf("模擬向第三方平台錢包 提幣")
				log.Printf("模擬送出 http.Post 向第三方平台錢包 提幣請求.... 假裝延遲3秒")

				time.Sleep(time.Second * 3) // 假裝延遲3秒

				if job.WaitQueueInfo != nil {

					log.Printf("組合回傳資料, 假裝有收到平台的用戶錢包數")
					result := map[string]interface{}{
						"message":       "success",
						"code":          0,
						"user":          job.UserId,
						"amount":        1000, // 假裝有收到平台回應的 1000元
						"transactionId": "平台給的交易Id",
					}

					if !job.WaitQueueInfo.IsCloseQueue {
						// 組合回傳資料
						receiveData := WaitQueueInfo{
							JobId: job.JobId,
							IsGet: true,
							Data:  result,
						}
						// 透過通道回傳給接收者
						job.WaitQueueInfo.WaitQueue <- receiveData
					}
				}

				log.Printf("結束處理 JobId:%v", job.JobId)

			case QUEUE_KIND_WALLET_WITHDARW:
				log.Printf("模擬向第三方平台錢包 存幣")

			case QUEUE_KIND_WALLET_AMOUNT_GET:
				log.Printf("取得用戶錢包數量 JobId:%v", job.JobId)

				log.Printf("模擬送出 http.Post 錢包數量請求.... 假裝延遲1秒")
				// http.Post ....
				time.Sleep(time.Second * 1) // 假裝延遲1秒

				if job.WaitQueueInfo != nil {

					log.Printf("組合回傳資料, 假裝有收到平台的用戶錢包數")
					result := map[string]interface{}{
						"message": "success",
						"code":    0,
						"user":    job.UserId,
						"amount":  1000, // 假裝有收到平台回應的 1000元
					}

					if !job.WaitQueueInfo.IsCloseQueue {
						// 組合回傳資料
						receiveData := WaitQueueInfo{
							JobId: job.JobId,
							IsGet: true,
							Data:  result,
						}
						// 透過通道回傳給接收者
						job.WaitQueueInfo.WaitQueue <- receiveData
					}
				}

				log.Printf("結束處理 JobId:%v", job.JobId)

			default:
				log.Printf("未知的動作 queueKind:%v", job.QueueKind)
			}
		}

		log.Printf("job通道監聽-離開")

	}()
}
