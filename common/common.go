package common

import (
	"fmt"
	"github.com/ontio/layer2deploy/layer2config"
	sdkcom "github.com/ontio/ontology-go-sdk/common"
	"github.com/ontio/ontology/common/log"
	"time"
)

const (
	WETHER_DATA_PROCESS string = "Weather Forecast"
)

func GetLayer2EventByTxHash(txHash string, cfg *layer2config.Config) (*sdkcom.SmartContactEvent, error) {
	var events *sdkcom.SmartContactEvent
	var err error
	var count uint32
	for {
		events, err = cfg.Layer2Sdk.GetSmartContractEvent(txHash)
		if err != nil {
			log.Errorf("GetLayer2EventByTxHash N.0 :%s\n", err)
			return nil, err
		}

		if events == nil && count < 30 {
			time.Sleep(time.Second)
			count++
			continue
		}

		break
	}

	if events != nil {
		if events.State == 0 {
			return nil, fmt.Errorf("error in events.State is 0 failed.")
		}

		return events, nil
	} else {
		return nil, fmt.Errorf("GetLayer2EventByTxHash failed counter over 30 times")
	}
}
