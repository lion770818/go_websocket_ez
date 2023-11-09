package ws

import (
	"errors"
	"log"
	"net/http"
	"runtime/debug"
	"server/src/util"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	ReadBuffMax  int = 1000
	WriteBuffMax int = 1000
)

type Connection struct {
	wsConnect *websocket.Conn // ws 資源
	inChan    chan []byte     // 讀 設置接收通道
	outChan   chan []byte     // 寫 設置寫入通道
	closeChan chan int32      // 關閉訊號通道

	mutex    sync.Mutex // 對closeChan關閉上鎖
	isClosed bool       // 防止closeChan被關閉多次

	ip       string
	socketId int
}

// 配置一個websocket連線
func NewConnection(wsConn *websocket.Conn) (conn *Connection) {

	conn = &Connection{
		wsConnect: wsConn,                          // ws 資源
		inChan:    make(chan []byte, ReadBuffMax),  // 讀 設置接收通道
		outChan:   make(chan []byte, WriteBuffMax), // 寫 設置寫入通道
		closeChan: make(chan int32, 1),             // 關閉訊號通道
	}

	log.Printf("配置一個websocket連線")
	return
}

// 啟動 讀寫 監聽 goroutine
func (conn *Connection) Run() {

	// 啟動讀 goroutine
	go conn.readLoop()
	// 啟動寫 goroutine
	go conn.writeLoop()
}

// 監聽 inChan 通道
func (conn *Connection) ReadMessage() (data []byte, err error) {

	// 接收各種通道的資料
	select {
	case data = <-conn.inChan: // 接收 inChan

	case <-conn.closeChan: // 接收停止訊號
		err = errors.New("connection is closeed readMessage")
	}
	return
}

// 將要寫給client的資料, 設置到outChan, 去依序寫入
func (conn *Connection) WriteMessage(data []byte) (err error) {

	select {
	case conn.outChan <- data:
	case <-conn.closeChan:
		err = errors.New("connection is closeed writeMessage")
	}
	return
}

func (conn *Connection) Close() {

	log.Printf("webscoket conn 資源釋放  socketId:%v", conn.socketId)

	if !conn.isClosed {
		return
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	conn.wsConnect.Close() // 關閉 ws 資源
	close(conn.inChan)     // 關閉通道
	close(conn.outChan)    // 關閉通道
	close(conn.closeChan)  // 關閉通道

	// 利用flag，讓closeChan只關閉一次
	conn.isClosed = true
}

// 讀取資料 goroutine
func (conn *Connection) readLoop() {

	defer func() {
		if err := recover(); err != nil {
			log.Printf("readLoop 引發例外 err:%v, memo:%v", err, string(debug.Stack()))
		}
	}()

	var (
		data []byte
		err  error
	)

	defer func() {
		log.Printf("close readLoop socketId:%v", conn.socketId)
		conn.ShutDown()
		conn.Close()

		log.Printf("exit readLoop....")
	}()

	for {

		// 設置讀取timeout時間, 超時就會切斷websocket 連線
		_ = conn.wsConnect.SetReadDeadline(time.Now().Add(time.Minute * 10))

		if _, data, err = conn.wsConnect.ReadMessage(); err != nil {
			log.Printf("websocket readMessage err=%v", err)
			return
		}

		_ = conn.wsConnect.SetReadDeadline(time.Time{})
		//阻塞在这里，等待inChan有空閒位置

		select {
		case conn.inChan <- data:
		case <-conn.closeChan: // closeChan 感知 conn 關閉
			log.Printf("closeChan tigger, exit...")
			return
		}

	}
}

// 寫入資料 goroutine
func (conn *Connection) writeLoop() {

	defer func() {
		if err := recover(); err != nil {
			log.Printf("writeLoop 引發例外 err:%v, memo:%v", err, string(debug.Stack()))
		}
	}()

	var (
		data []byte
		err  error
	)

	defer func() {
		log.Printf("close writeLoop socketId:%v", conn.socketId)
		conn.ShutDown()
		conn.Close()

		log.Printf("exit writeLoop....")
	}()

	for {
		select {
		case data = <-conn.outChan:
		case <-conn.closeChan:
			log.Printf("websocket closeChan quit writeloop")
			return
		}
		if err = conn.wsConnect.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("websocket writeMessage err=%v", err)
			return
		}
	}

}

// 主動關閉用戶
func (conn *Connection) ShutDown() {
	conn.closeChan <- 1
}

// 取得用戶連線IP
func (conn *Connection) GetIP(req *http.Request) (string, int64) {

	// proxy 轉發
	clientIpStr := req.Header.Get("X-Forwarded-For")
	// 一般直連的 IP
	if len(clientIpStr) == 0 {
		clientIpStr = req.RemoteAddr
	}
	// 原本寫法
	if len(clientIpStr) == 0 {
		clientIpStr = req.Host
	}

	// 取出 socketID
	ipStr := strings.Split(clientIpStr, ":")
	var socketID int64
	var clientIP string
	if len(ipStr) > 1 {
		socketID = util.Str2Int64(ipStr[1])

		// IP + socketID 取ipStr[0]
		clientIP = ipStr[0]
	} else {
		// 只有一個參數
		clientIP = clientIpStr
	}

	return clientIP, socketID
}
