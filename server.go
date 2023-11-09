// server.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"server/src/jobqueue"
	"server/src/transaction"
	"server/src/util"
	ws "server/src/websocket"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		// 允許跨域
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type User struct {
	UserId  int64
	QueueMg *jobqueue.QueueWorkManager
}

func userHandler(w http.ResponseWriter, r *http.Request) {

	defer func() {
		if err := recover(); err != nil {
			log.Printf("userHandler 引發例外 err:%v, memo:%v", err, string(debug.Stack()))
		}
	}()

	// Upgrade our raw HTTP connection to a websocket based one
	websocket_conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error during connection upgradation:", err)
		return
	}
	defer websocket_conn.Close()

	// 建立 ws 連線資源
	conn := ws.NewConnection(websocket_conn)
	// 取得 ip和 socketId
	ip, socketId := conn.GetIP(r)
	// 啟動 讀寫 監聽 goroutine
	conn.Run()

	var exit bool
	var playCnt int
	var platformID int = 1

	// 配置登入的使用者
	user := User{
		UserId:  socketId,               // 假裝 userId = socketId
		QueueMg: jobqueue.NewJobQueue(), // 每個使用者配發一個專屬通道, 去跟第三方wallet溝通
	}
	user.QueueMg.Run() // 開始監聽用戶工作佇列
	log.Printf("connect success ip:%v socketId:%v", ip, socketId)

	for {

		if exit {
			log.Printf("exit main for loop")
			break
		}

		log.Printf("持續監聽 inChan 通道, 如果有通道有收到資料 會返回read_data")
		read_data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read err=%v", err)
			exit = true
		} else {

			log.Printf("read_data=%v", string(read_data[:]))

			// json to map
			m, err := util.JsonToMap(string(read_data[:]))
			if err != nil {
				log.Printf("err:%v", err)
			}

			// 解析封包
			cmd, ok := m["cmd"]
			if !ok {
				exit = true
			}
			data, ok := m["data"]
			if !ok {
				exit = true
			}
			switch cmd {
			case "login":
				log.Printf("收到登入封包 data:%v", data)

				// TODO:驗證

				// TODO:讀取資料庫, 尋找是否有此會員

				// TODO:新增 user 到 Redis

				// 向錢包進行提款動作
				data := map[string]interface{}{
					"userId": user.UserId, // 使用socketId 暫時當作用戶的UserId
				}
				// 寫入工作jueue (取得用戶錢包數量)
				waitinfo := user.QueueMg.Insert(platformID, user.UserId, "", jobqueue.QUEUE_KIND_WALLET_AMOUNT_GET, true, data)
				// 等待接收結果
				userData := user.QueueMg.WaitQueue(waitinfo)
				log.Printf("amount get userData:%+v", userData)

				amount := userData.Data["amount"]

				// 產生交易Id
				transactionId := transaction.InstanceGet().MakeId(platformID, user.UserId, "deposit")
				data = map[string]interface{}{
					"userId":        user.UserId,   // 使用socketId 暫時當作用戶的UserId
					"amount":        amount,        // 提幣數量
					"transactionId": transactionId, // 交易Id
				}

				// TODO:新增 交易資訊 到 Redis
				// 開始提幣交易
				waitinfo = user.QueueMg.Insert(platformID, user.UserId, transactionId, jobqueue.QUEUE_KIND_WALLET_DEPOSIT, true, data)
				// 等待接收結果
				userData = user.QueueMg.WaitQueue(waitinfo)
				log.Printf("deposit waitinfo:%+v", waitinfo)

				// 組合回傳資料
				result := map[string]interface{}{

					"message": "success",
					"code":    0,
					"cmd":     cmd,
					"data":    userData,
				}

				// 回傳給client
				log.Printf("result:%+v", result)
				conn.WriteMessage(util.MapToJsonByte(result))
			case "play":
				log.Printf("收到遊戲封包 data:%v", data)

				// 假設贏得10分
				win := 10

				// 遊戲次數增加
				playCnt++
				data := map[string]interface{}{
					"userId":  user.UserId, // 使用socketId 暫時當作用戶的UserId
					"win":     win,
					"playCnt": playCnt,
				}
				result := map[string]interface{}{
					"cmd":     cmd,
					"message": "success",
					"code":    0,
					"data":    data,
				}

				// 回傳給client
				log.Printf("result:%v", result)
				conn.WriteMessage(util.MapToJsonByte(result))

			case "close_server":
				log.Printf("收到關閉封包 data:%v", data)

				// 模擬關閉某個連線, 實務上應該是後台打http api 來關閉某個用戶連線
				conn.ShutDown()
			default:
				log.Printf("unknow cmd:%s", cmd)
				exit = true
			}

		}
	}

	log.Printf("exit....")
}

func main() {

	port := "8080"
	log.Printf("welcome websocket example... v1")

	log.Printf("wesocket url ws://localhost:%v/user", port)

	http.HandleFunc("/user", userHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%s", port), nil))
}
