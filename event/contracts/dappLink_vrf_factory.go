package contracts

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/the-web3/dapplink-vrf/bindings"
	"github.com/the-web3/dapplink-vrf/database"
	"github.com/the-web3/dapplink-vrf/database/event"
	"github.com/the-web3/dapplink-vrf/database/worker"
	"github.com/the-web3/dapplink-vrf/synchronizer/retry"
)

type DappLinkVrfFactory struct {
	DlVrfFactoryAbi    *abi.ABI
	DlVrfFactoryFilter *bindings.DappLinkVRFFactoryFilterer
}

func NewDappLinkVrfFactory() (*DappLinkVrfFactory, error) {
	dappLinkVrfFactoryAbi, err := bindings.DappLinkVRFFactoryMetaData.GetAbi()
	if err != nil {
		log.Error("get dapplink vrf factory abi fail", "err", err)
		return nil, err
	}
	dappLinkVrfFactoryFilterer, err := bindings.NewDappLinkVRFFactoryFilterer(common.Address{}, nil)
	if err != nil {
		log.Error("new dapplink vrf factory filter fail", "err", err)
		return nil, err
	}
	return &DappLinkVrfFactory{
		DlVrfFactoryAbi:    dappLinkVrfFactoryAbi,
		DlVrfFactoryFilter: dappLinkVrfFactoryFilterer,
	}, nil
}

func (dvff *DappLinkVrfFactory) ProcessDappLinkVrfFactoryEvent(db *database.DB, dappLinkVrfFactoryAddres string, startHeight, endHeight *big.Int) error {
	contactFilter := event.ContractEvent{ContractAddress: common.HexToAddress(dappLinkVrfFactoryAddres)}
	contractEventList, err := db.ContractEvent.ContractEventsWithFilter(contactFilter, startHeight, endHeight)
	if err != nil {
		log.Error("query contacts event fail", "err", err)
		return err
	}
	var proxyCreatedList []worker.PoxyCreated
	for _, contractEvent := range contractEventList {
		if contractEvent.EventSignature.String() == dvff.DlVrfFactoryAbi.Events["ProxyCreated"].ID.String() {
			proxyCreated, err := dvff.DlVrfFactoryFilter.ParseProxyCreated(*contractEvent.RLPLog)
			if err != nil {
				log.Error("proxy created fail", "err", err)
				return err
			}
			log.Info("proxy created event", "MintProxyAddress", proxyCreated.MintProxyAddress)
			pc := worker.PoxyCreated{
				GUID:         uuid.New(),
				ProxyAddress: proxyCreated.MintProxyAddress,
				Timestamp:    uint64(time.Now().Unix()),
			}
			proxyCreatedList = append(proxyCreatedList, pc)
		}
	}

	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
	if _, err := retry.Do[interface{}](context.Background(), 10, retryStrategy, func() (interface{}, error) {
		if err := db.Transaction(func(tx *database.DB) error {
			if err := tx.PoxyCreated.StorePoxyCreated(proxyCreatedList); err != nil {
				return err
			}
			return nil
		}); err != nil {
			log.Info("unable to persist batch", err)
			return nil, fmt.Errorf("unable to persist batch: %w", err)
		}
		return nil, nil
	}); err != nil {
		return err
	}
	return nil
}
