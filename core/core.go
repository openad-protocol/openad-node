package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ontio/layer2deploy/http"
	"github.com/ontio/layer2deploy/layer2config"
	ontSdk "github.com/ontio/ontology-go-sdk"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/patrickmn/go-cache"
)

type RequestLog struct {
	Timestamp       time.Time     `json:"time"`
	ResponseStatus  int64         `json:"status"`
	ProcessDuration time.Duration `json:"process_duration"`
	HTTPMethod      string        `json:"http_method"`
	Path            string        `json:"path"`
	RequestParam    string        `json:"request_param"`
	ResponseParam   string        `json:"response_param"`
}

type SendService struct {
	Wg      *sync.WaitGroup
	QuitS   chan bool
	Cfg     *layer2config.Config
	Enabled uint32
}

var DefSendService *SendService

func NewSendService(cfg *layer2config.Config) *SendService {
	return &SendService{
		Wg:      new(sync.WaitGroup),
		QuitS:   make(chan bool),
		Cfg:     cfg,
		Enabled: 1, //default enabled
	}
}

func (self *SendService) RepeantSendLogToChain() {
	self.Wg.Add(1)
	defer self.Wg.Done()

	log.Infof("RepeantSendLogToChain Y.0. start to send log to chain.")
	for {
		atomic.LoadUint32(&self.Enabled)

		select {
		case <-self.QuitS:
			log.Infof("RepeantSendLogToChain get QuitS signal")
			return
		default:
			if self.Enabled == 0 {
				log.Infof("RepeantSendLogToChain Disabled")
				time.Sleep(time.Second * time.Duration(self.Cfg.Layer2RecordInterval*10))
				continue
			}
			time.Sleep(time.Second * time.Duration(self.Cfg.Layer2RecordInterval))
		}
	}
}

func GetCommitedLayer2Height(ontSdk *ontSdk.OntologySdk, contract common.Address) (uint32, error) {
	tx, err := ontSdk.NeoVM.NewNeoVMInvokeTransaction(0, 0, contract, []interface{}{"getCurrentHeight", []interface{}{}})
	if err != nil {
		return 0, err
	}
	result, err := ontSdk.PreExecTransaction(tx)
	if err != nil {
		fmt.Printf("PreExecTransaction failed! err: %s", err.Error())
		return 0, err
	}
	if result == nil {
		fmt.Printf("can not find the result")
		return 0, fmt.Errorf("can not find current height!")
	}
	height, err := result.Result.ToInteger()
	if err != nil {
		return 0, fmt.Errorf("current height is not right!")
	}
	return uint32(height.Uint64()), nil
}

func GetCommitedLayer2StateByHeight(ontSdk *ontSdk.OntologySdk, contract common.Address, height uint32) ([]byte, uint32, error) {
	tx, err := ontSdk.NeoVM.NewNeoVMInvokeTransaction(0, 0, contract, []interface{}{"getStateRootByHeight", []interface{}{height}})
	if err != nil {
		fmt.Printf("new transaction failed!")

	}
	result, err := ontSdk.PreExecTransaction(tx)
	if err != nil {
		fmt.Printf("PreExecTransaction failed! err: %s", err.Error())
		return nil, 0, err

	}
	if result == nil {
		fmt.Printf("can not find the result")
		return nil, 0, fmt.Errorf("can not find state of heigh: %d", height)

	}
	tt, _ := result.Result.ToArray()
	if len(tt) != 3 {
		fmt.Printf("result is not right")
		return nil, 0, fmt.Errorf("result is not right, height: %d", height)

	}
	item0, _ := tt[0].ToString()
	item1, _ := tt[1].ToInteger()
	item2, _ := tt[2].ToInteger()
	fmt.Printf("item0: %s, item1: %d, item2: %d\n", item0, item1, item2)
	stateRoot, err := common.Uint256FromHexString(item0)
	if err != nil {
		return nil, 0, fmt.Errorf("state hash is not right, height: %d", height)

	}
	return stateRoot.ToArray(), uint32(item1.Uint64()), nil
}

func CheckHashExistInlayer2(layer2Sdk *ontSdk.Layer2Sdk, contract common.Address, hash []byte) (bool, error) {
	tx, err := layer2Sdk.NeoVM.NewNeoVMInvokeTransaction(0, 0, contract, []interface{}{"CheckHashExist", []interface{}{hash}})
	if err != nil {
		fmt.Printf("new transaction failed!")

	}
	result, err := layer2Sdk.PreExecTransaction(tx)
	if err != nil {
		fmt.Printf("PreExecTransaction failed! err: %s", err.Error())
		return false, err
	}

	if result == nil {
		return false, fmt.Errorf("can not find the result")
	}

	tt, err := result.Result.ToInteger()
	if tt.Int64() == 0 {
		return false, nil
	} else {
		return true, nil
	}
}

type VerifyService struct {
	Cfg          *layer2config.Config
	VResultCache *cache.Cache
	VerifyLock   *sync.Mutex
	HttpClient   *http.Client
}

const (
	CACHE_SIZE = 1000
)

func NewVerifyService(cfg *layer2config.Config) *VerifyService {
	return &VerifyService{
		Cfg:          cfg,
		VResultCache: cache.New(240*time.Hour, 240*time.Hour),
		VerifyLock:   new(sync.Mutex),
		HttpClient:   http.NewClient(),
	}
}

var DefVerifyService *VerifyService

const (
	SUCCESS          = 0
	PROCESSING       = 1
	FAILED           = 2
	NORECORDONLAYER2 = 3
)

type VerifyResult struct {
	Code             int32  `json:"code"`
	FailedMsg        string `json:"failedMsg"`
	Key              string `json:"key"`
	Value            string `json:"value"`
	Proof            string `json:"proof"`
	Layer2Height     uint32 `json:"layer2Height"`
	CommitHeight     uint32 `json:"commitHeight"`
	WitnessStateRoot string `json:"witnessStateRoot"`
	WitnessContract  string `json:"witnessContract"`
	Layer1Tx         string `json:"layer1Tx"`
}

type ContractTxResp struct {
	Code   int               `json:"code"`
	Msg    string            `json:"msg"`
	Result *ContractTxResult `json:"result"`
}

type ContractTxResult struct {
	Total   int                 `json:"total"`
	Records []*ContractTxRecord `json:"records"`
}

type ContractTxRecord struct {
	TxHash      string `json:"tx_hash"`
	TxType      int    `json:"tx_type"`
	TxTime      int    `json:"tx_time"`
	BlockHeight int    `json:"block_height"`
	BlockIndex  int    `json:"block_index"`
	Fee         string `json:"fee"`
	ConfirmFlag int    `json:"confirm_flag"`
}

// verify the store
func (self *VerifyService) StoreHashCore(hash []byte) (string, error) {
	layer2Sdk := self.Cfg.Layer2Sdk
	contractAddress, err := common.AddressFromHexString(self.Cfg.Layer2Contract)
	if err != nil {
		return "", err
	}
	txHash, err := layer2Sdk.NeoVM.InvokeNeoVMContract(uint64(self.Cfg.GasPrice), 200000, nil, self.Cfg.AdminAccount, contractAddress, []interface{}{"StoreHash", []interface{}{hash}})
	if err != nil {
		log.Errorf("StoreHashCore failed")
		return "", err
	}

	return txHash.ToHexString(), nil
}

// verify the store
func (self *VerifyService) VerifyHashCore(hash string) (*VerifyResult, error) {
	self.VerifyLock.Lock()
	defer self.VerifyLock.Unlock()
	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return nil, err
	}
	contractAddr, err := common.AddressFromHexString(self.Cfg.Layer2Contract)
	if err != nil {
		return nil, err
	}

	data, found := self.VResultCache.Get(hash)
	if !found {
		recordOnLayer2, err := CheckHashExistInlayer2(self.Cfg.Layer2Sdk, contractAddr, hashBytes)
		if err != nil {
			return nil, err
		}
		if !recordOnLayer2 {
			result := &VerifyResult{
				Code:      NORECORDONLAYER2,
				FailedMsg: "have not record on layer2",
			}
			log.Infof("VerifyHashCore not found %s", hash)
			return result, nil
		}

		result := &VerifyResult{
			Code:      PROCESSING,
			FailedMsg: "still processing. please wait.",
		}
		self.VResultCache.Set(hash, result, cache.DefaultExpiration)

		go self.AsyncVerifyResult(hash)
		return result, nil
	} else {
		result := data.(*VerifyResult)
		switch result.Code {
		case SUCCESS, PROCESSING:
			return result, nil
		case FAILED:
			log.Infof("VerifyHashCore %s failed before %v", hash, *result)
			result := &VerifyResult{
				Code:      PROCESSING,
				FailedMsg: "still processing. please wait.",
			}
			self.VResultCache.Set(hash, result, cache.DefaultExpiration)

			go self.AsyncVerifyResult(hash)
			return result, nil
		default:
			return nil, fmt.Errorf("get wrong result code %d", result.Code)
		}
	}
}

func (self *VerifyService) AsyncVerifyResult(hash string) {
	var err error
	layer2Sdk := self.Cfg.Layer2Sdk
	ontSdk := self.Cfg.OntSdk
	var result *VerifyResult
	defer func() {
		if err != nil {
			result = &VerifyResult{
				Code:      FAILED,
				FailedMsg: fmt.Sprintf("%s", err),
			}
		}
		self.VResultCache.Set(hash, result, cache.DefaultExpiration)
		log.Infof("AsyncVerifyResult Y.0 %v", *result)
	}()

	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		log.Debugf("VerifyHash: N.0 %s", err)
		return
	}
	// 1. get the store key
	//    get the store data, store proof by the key
	key, _ := layer2Sdk.GetLayer2StoreKey(self.Cfg.Layer2Contract, hashBytes)
	store, err := layer2Sdk.GetLayer2StoreProof(key)
	if err != nil {
		log.Errorf("AsyncVerifyResult N.0 key: %s. %s", hash, err)
		return
	}

	log.Infof("verify key: %s , value: %s, proof: %s, layer2height: %d", hash, store.Value, store.Proof, store.Height)

	// 2. ensure the state root of the store is commited to ontology
	contractAddress, err := common.AddressFromHexString(self.Cfg.Layer2WitContract)
	if err != nil {
		log.Errorf("AsyncVerifyResult N.1 key: %s. %s", hash, err)
		return
	}

	var count uint32
	for {
		if count > uint32(self.Cfg.Layer2RetryCount) {
			err = fmt.Errorf("AsyncVerifyResult Retry over times")
			return
		}

		count++
		curHeight, err := GetCommitedLayer2Height(ontSdk, contractAddress)
		if err != nil {
			log.Errorf("AsyncVerifyResult N.3 key: %s. %s", hash, err)
			return
		}

		if curHeight < store.Height {
			log.Infof("AsyncVerifyResult N.3.0 : %s.  wait layer2 relayer commit layer2 block to height %d. currHeight: %d", hash, store.Height, curHeight)
			result := &VerifyResult{
				Code:      PROCESSING,
				FailedMsg: fmt.Sprintf("wait layer2 relayer commit layer2 block to height %d. currHeight: %d", store.Height, curHeight),
			}
			self.VResultCache.Set(hash, result, cache.DefaultExpiration)

			time.Sleep(time.Second * time.Duration(self.Cfg.Layer2RecordInterval))
			continue
		}
		break
	}

	// get layer1 transaction
	time.Sleep(time.Second * time.Duration(self.Cfg.Layer2RecordInterval))
	var layer1TxHash = ""
	i := 1
	for i < 6 {
		var url = self.Cfg.ExplorerUrl + "/v2/contracts/other/" + self.Cfg.Layer2WitContract + "/transactions?page_size=10&page_number=" + strconv.Itoa(i)
		resp, err := self.HttpClient.Get(url)
		if err != nil {
			log.Errorf("%s HttpClient get error", url, err)
			return
		}
		contractResp := &ContractTxResp{}
		err = json.Unmarshal(resp, &contractResp)
		if err != nil {
			log.Errorf("Unmarshal explorer result error")
			return
		}
		records := contractResp.Result.Records
		i++
	out:
		for _, record := range records {
			txHash := record.TxHash
			event, err := self.Cfg.OntSdk.GetSmartContractEvent(txHash)
			if err != nil {
				log.Errorf("GetSmartContractEvent error")
				return
			}
			notifies := event.Notify
			for _, notify := range notifies {
				contractAddress := notify.ContractAddress
				if self.Cfg.Layer2WitContract == contractAddress {
					states, ok := notify.States.([]interface{})
					if !ok {
						continue
					}
					height, ok := states[2].(string)
					if ok {
						decodeString, err := hex.DecodeString(height)
						if err != nil {
							log.Errorf("DecodeString error")
							return
						}
						layer2Height := common.BigIntFromNeoBytes(decodeString)
						if layer2Height.Uint64() == uint64(store.Height) {
							layer1TxHash = event.TxHash
							i = 100
							break out
						}
					}
				}
			}
		}
	}

	// 3. get the state root which is commited to ontology
	stateRoot, height, err := GetCommitedLayer2StateByHeight(ontSdk, contractAddress, store.Height)
	if err != nil {
		log.Errorf("AsyncVerifyResult N.4 key: %s. %s", hash, err)
		return
	}
	//log.Infof("state root: %s, height: %d\n", hex.EncodeToString(stateRoot), height)

	// 4. verify the data is stored through the store proof and state root
	proof_byte, err := hex.DecodeString(store.Proof)
	if err != nil {
		log.Errorf("AsyncVerifyResult N.5 key: %s. %s", hash, err)
		return
	}
	value_bytes, err := hex.DecodeString(store.Value)
	if err != nil {
		log.Errorf("AsyncVerifyResult N.6 key: %s. %s", hash, err)
		return
	}
	exist, err := layer2Sdk.VerifyLayer2StoreProof(key, value_bytes, proof_byte, stateRoot)
	if err != nil {
		log.Errorf("AsyncVerifyResult N.7 key: %s. %s", hash, err)
		return
	}
	if !exist {
		err = fmt.Errorf("verify failed")
		return
	}

	result = &VerifyResult{
		Code:             SUCCESS,
		Key:              hash,
		Value:            store.Value,
		Proof:            store.Proof,
		Layer2Height:     store.Height,
		CommitHeight:     height,
		WitnessStateRoot: hex.EncodeToString(stateRoot),
		WitnessContract:  self.Cfg.Layer2WitContract,
		Layer1Tx:         layer1TxHash,
	}

	return
}
