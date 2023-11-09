package ws

import (
	"errors"
	"log"
	"runtime/debug"
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
}

// 配置一個websocket連線
func NewConnection(wsConn *websocket.Conn) (conn *Connection) {

	conn = &Connection{
		wsConnect: wsConn,                          // ws 資源
		inChan:    make(chan []byte, ReadBuffMax),  // 讀 設置接收通道
		outChan:   make(chan []byte, WriteBuffMax), // 寫 設置寫入通道
		closeChan: make(chan int32, 1),             // 關閉訊號通道
	}

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

	conn.wsConnect.Close() // 關閉 ws 資源

	// 利用flag，讓closeChan只關閉一次
	conn.mutex.Lock()
	if !conn.isClosed {
		close(conn.closeChan) // 關閉通道
		conn.isClosed = true
	}
	conn.mutex.Unlock()
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
		log.Printf("close readLoop")
		conn.Close()
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
		log.Printf("close writeLoop")
		conn.Close()
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

func (conn *Connection) ShutDown() {
	conn.closeChan <- 1
}
