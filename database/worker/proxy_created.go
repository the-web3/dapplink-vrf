package worker

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PoxyCreated struct {
	GUID         uuid.UUID      `gorm:"primaryKey" json:"guid"`
	ProxyAddress common.Address `json:"proxy_address" gorm:"serializer:bytes"`
	Timestamp    uint64
}

type PoxyCreatedView interface {
}

type PoxyCreatedDB interface {
	PoxyCreatedView

	StorePoxyCreated([]PoxyCreated) error
}

type poxyCreatedDB struct {
	gorm *gorm.DB
}

func NewPoxyCreatedDB(db *gorm.DB) PoxyCreatedDB {
	return &poxyCreatedDB{gorm: db}
}

func (db poxyCreatedDB) StorePoxyCreated(PoxyCreatedList []PoxyCreated) error {
	result := db.gorm.Table("proxy_created").CreateInBatches(&PoxyCreatedList, len(PoxyCreatedList))
	return result.Error
}
