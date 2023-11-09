package transaction

import (
	"fmt"
	"sync"
	"time"
)

type TransactionManager struct {
	TransactionCount int64 // 交易流水id TODO:可以使用mysql AUTO_INCREMENT 數值, 這裡先簡單計數就好
	mutex            sync.Mutex
}

var transactionMg *TransactionManager

func InstanceGet() *TransactionManager {
	if transactionMg == nil {

		transactionMg = &TransactionManager{}

	}
	return transactionMg
}

func (manager *TransactionManager) MakeId(platformID int, userId int64, transactionType string) string {

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	now := time.Now()
	timeStr := fmt.Sprintf("%d%d%d", now.Year(), now.Month(), now.Day())

	// 組合交易Id
	transferID := fmt.Sprintf("%d-%d-%s-%s-%010d",
		platformID, userId, transactionType, timeStr, manager.TransactionCount)

	// 流水號+1
	manager.TransactionCount++

	return transferID
}
