package worker

import (
	"gorm.io/gorm"
	"math/big"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/common"
)

type RequestSend struct {
	GUID       uuid.UUID      `gorm:"primaryKey" json:"guid"`
	RequestId  *big.Int       `json:"request_id" gorm:"serializer:u256"`
	VrfAddress common.Address `json:"vrf_address" gorm:"serializer:bytes"`
	NumWords   *big.Int       `json:"num_words" gorm:"serializer:u256"` // 0:扫到合约事件,1:充值成功
	Status     uint8          `json:"status"`                           // 0:扫到合约事件,1:充值成功
	Timestamp  uint64
}

type RequestSendView interface {
}

type RequestSendDB interface {
	RequestSendView

	StoreRequestSend([]RequestSend) error
}

type requestSendDB struct {
	gorm *gorm.DB
}

func NewRequestSendDB(db *gorm.DB) RequestSendDB {
	return &requestSendDB{gorm: db}
}

func (db requestSendDB) StoreRequestSend(RequestSendList []RequestSend) error {
	result := db.gorm.CreateInBatches(&RequestSendList, len(RequestSendList))
	return result.Error
}
