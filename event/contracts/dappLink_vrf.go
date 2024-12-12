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

type DappLinkVrf struct {
	DlVrfAbi    *abi.ABI
	DlVrfFilter *bindings.DappLinkVRFFilterer
}

func NewDappLinkVrf() (*DappLinkVrf, error) {
	dappLinkVrfAbi, err := bindings.DappLinkVRFMetaData.GetAbi()
	if err != nil {
		log.Error("get dapplink vrf abi fail", "err", err)
		return nil, err
	}
	dappLinkVRFFilterer, err := bindings.NewDappLinkVRFFilterer(common.Address{}, nil)
	if err != nil {
		log.Error("new dapplink vrf filter fail", "err", err)
		return nil, err
	}
	return &DappLinkVrf{
		DlVrfAbi:    dappLinkVrfAbi,
		DlVrfFilter: dappLinkVRFFilterer,
	}, nil
}

func (dvf *DappLinkVrf) ProcessDappLinkVrfEvent(db *database.DB, dappLinkVrfAddres string, startHeight, endHeight *big.Int) error {
	contactFilter := event.ContractEvent{ContractAddress: common.HexToAddress(dappLinkVrfAddres)}
	contractEventList, err := db.ContractEvent.ContractEventsWithFilter(contactFilter, startHeight, endHeight)
	if err != nil {
		log.Error("query contacts event fail", "err", err)
		return err
	}
	var RequestSentList []worker.RequestSend
	var FillRandomWordList []worker.FillRandomWords
	for _, contractEvent := range contractEventList {
		if contractEvent.EventSignature.String() == dvf.DlVrfAbi.Events["RequestSent"].ID.String() {
			rquestSentEvent, err := dvf.DlVrfFilter.ParseRequestSent(*contractEvent.RLPLog)
			if err != nil {
				log.Error("parse request sent fail", "err", err)
				return err
			}
			log.Info("Request sent event", "RequestId", rquestSentEvent.RequestId, "NumWords", rquestSentEvent.NumWords, "Current", rquestSentEvent.Current)
			rs := worker.RequestSend{
				GUID:       uuid.New(),
				RequestId:  rquestSentEvent.RequestId,
				VrfAddress: rquestSentEvent.Current,
				NumWords:   rquestSentEvent.NumWords,
				Status:     0,
				Timestamp:  uint64(time.Now().Unix()),
			}
			RequestSentList = append(RequestSentList, rs)
		}

		if contractEvent.EventSignature.String() == dvf.DlVrfAbi.Events["FillRandomWords"].ID.String() {
			fillRandomWords, err := dvf.DlVrfFilter.ParseFillRandomWords(*contractEvent.RLPLog)
			if err != nil {
				log.Error("parse fill random fail", "err", err)
				return err
			}
			log.Info("Fill random words event", "RequestId", fillRandomWords.RequestId, "RandomWords", fillRandomWords.RandomWords)
			var randomWords string
			for _, rword := range fillRandomWords.RandomWords {
				randomWords = rword.String()
			}
			frw := worker.FillRandomWords{
				GUID:        uuid.New(),
				RequestId:   fillRandomWords.RequestId,
				RandomWords: randomWords,
				Timestamp:   uint64(time.Now().Unix()),
			}
			FillRandomWordList = append(FillRandomWordList, frw)
		}
	}

	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
	if _, err := retry.Do[interface{}](context.Background(), 10, retryStrategy, func() (interface{}, error) {
		if err := db.Transaction(func(tx *database.DB) error {
			if err := tx.FillRandomWords.StoreFillRandomWords(FillRandomWordList); err != nil {
				return err
			}
			if err := tx.RequestSend.StoreRequestSend(RequestSentList); err != nil {
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
