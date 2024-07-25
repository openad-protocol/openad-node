package cmd

import (
	"encoding/json"
	"fmt"
	common2 "github.com/ontio/layer2deploy/common"
	"github.com/ontio/layer2deploy/layer2config"
	ontSdk "github.com/ontio/ontology-go-sdk"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/common/password"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
)

func SetLayer2Config(ctx *cli.Context) (*layer2config.Config, error) {
	cf := ctx.String(GetFlagName(ConfigfileFlag))
	if _, err := os.Stat(cf); os.IsNotExist(err) {
		// if there's no config file, use default config
		return nil, err
	}

	file, err := os.Open(cf)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bs, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	cfg := &layer2config.Config{}
	err = json.Unmarshal(bs, cfg)
	if err != nil {
		return nil, err
	}

	if cfg.WalletName == "" || cfg.Layer2MainNetNode == "" || cfg.Layer2TestNetNode == "" {
		return nil, fmt.Errorf("walletName/layer2MainNetAddress/layer2TestNetAddress  is nil")
	}

	// if cfg.RequestLogServer == "" || cfg.Layer2RecordInterval == 0 || cfg.Layer2RetryCount == 0 || cfg.Layer2RecordBatchC == 0 {
	// 	return nil, fmt.Errorf("RequestLogServer/Layer2RecordInterval/Layer2RetryCount/Layer2RecordBatchC config error")
	// }

	wallet, err := ontSdk.OpenWallet(cfg.WalletName)
	if err != nil {
		return nil, err
	}
	passwd, err := password.GetAccountPassword()
	if err != nil {
		return nil, err
	}
	sagaAccount, err := wallet.GetDefaultAccount(passwd)
	if err != nil {
		return nil, err
	}

	layer2Sdk := ontSdk.NewLayer2Sdk()
	ontoSdk := ontSdk.NewOntologySdk()
	switch cfg.NetWorkId {
	case layer2config.NETWORK_ID_MAIN_NET:
		log.Infof("currently Main net")
		layer2Sdk.NewRpcClient().SetAddress(cfg.Layer2MainNetNode)
		ontoSdk.NewRpcClient().SetAddress("http://dappnode3.ont.io:20336")
	case layer2config.NETWORK_ID_POLARIS_NET:
		log.Infof("currently test net")
		layer2Sdk.NewRpcClient().SetAddress(cfg.Layer2TestNetNode)
		ontoSdk.NewRpcClient().SetAddress("http://polaris4.ont.io:20336")
	case layer2config.NETWORK_ID_SOLO_NET:
		log.Infof("currently solo net")
		// solo simulation with test net. but different contract and owner
		layer2Sdk.NewRpcClient().SetAddress(cfg.Layer2TestNetNode)
		ontoSdk.NewRpcClient().SetAddress("http://polaris4.ont.io:20336")
	default:
		return nil, fmt.Errorf("error network id %d", cfg.NetWorkId)
	}

	cfg.Layer2Sdk = layer2Sdk
	cfg.OntSdk = ontoSdk
	cfg.AdminAccount = sagaAccount
	err = CheckLayer2InitAddress(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func CheckLayer2InitAddress(cfg *layer2config.Config) error {
	layer2Sdk := cfg.Layer2Sdk
	if layer2Sdk == nil {
		return fmt.Errorf("layer2 sdk should not be nil")
	}

	if len(cfg.Layer2Contract) == 0 {
		return fmt.Errorf("layer2 contract address or contract not init")
	}

	log.Infof("layer2Contract %s", cfg.Layer2Contract)

	contractAddr, err := common.AddressFromHexString(cfg.Layer2Contract)
	//incase that is a file name.
	//if err != nil || len(cfg.Layer2Contract) != common.ADDR_LEN*2 {
	//	code, err := ioutil.ReadFile(cfg.Layer2Contract)
	//	if err != nil {
	//		return fmt.Errorf("error in ReadFile: %s, %s\n", cfg.Layer2Contract, err)
	//	}
	//
	//	codeHash := common.ToHexString(code)
	//	contractAddr, err = utils.GetContractAddress(codeHash)
	//	if err != nil {
	//		return fmt.Errorf("error get contract address %s", err)
	//	}
	//
	//	payload, err := layer2Sdk.GetSmartContract(contractAddr.ToHexString())
	//	if payload == nil || err != nil {
	//		txHash, err := layer2Sdk.NeoVM.DeployNeoVMSmartContract(uint64(cfg.GasPrice), 200000000, cfg.AdminAccount, true, codeHash, "witness layer2 contract", "1.0", "witness", "email", "desc")
	//		if err != nil {
	//			return fmt.Errorf("deploy contract %s err: %s", cfg.Layer2Contract, err)
	//		}
	//
	//		_, err = common2.GetLayer2EventByTxHash(txHash.ToHexString(), cfg)
	//		if err != nil {
	//			return fmt.Errorf("deploy contract failed %s", err)
	//		}
	//		log.Infof("deploy concontract success")
	//	}
	//
	//	log.Infof("the contractAddr hexstring is %s", contractAddr.ToHexString())
	//	cfg.Layer2Contract = contractAddr.ToHexString()
	//}

	//contractAddr, err = common.AddressFromHexString(cfg.Layer2Contract)
	//if err != nil {
	//	return err
	//}

	for {
		res, err := layer2Sdk.NeoVM.PreExecInvokeNeoVMContract(contractAddr, []interface{}{"init_status", []interface{}{}})
		if err != nil {
			return fmt.Errorf("err get init_status %s", err)
		}

		if res.State == 0 {
			return fmt.Errorf("init statuc exec failed state is 0")
		}

		addrB, err := res.Result.ToByteArray()
		if err != nil {
			return fmt.Errorf("error init_status toByteArray %s", err)
		}

		sagaAddrBase58 := cfg.AdminAccount.Address.ToBase58()
		if len(addrB) != 0 {
			addrO, err := common.AddressParseFromBytes(addrB)
			if err != nil {
				return fmt.Errorf("AddressParseFromBytes err: %s", err)
			}

			log.Infof("layer2 address already init owner to addr %s", addrO.ToBase58())
			if addrO.ToBase58() != sagaAddrBase58 {
				return fmt.Errorf("contract addr not equal. owner is %s. but sagaAccount init to %s", addrO.ToBase58(), sagaAddrBase58)
			}
			break
		} else {
			log.Infof("start init layer2 addr owner to address %s", sagaAddrBase58)
			txHash, err := layer2Sdk.NeoVM.InvokeNeoVMContract(uint64(cfg.GasPrice), 200000, nil, cfg.AdminAccount, contractAddr, []interface{}{"init", cfg.AdminAccount.Address})
			if err != nil {
				return fmt.Errorf("init layer2 owner err0 %s", err)
			}
			_, err = common2.GetLayer2EventByTxHash(txHash.ToHexString(), cfg)
			if err != nil {
				return fmt.Errorf("init layer2 owner err1: %s", err)
			}
			log.Infof("init layer2 addr owner to address %s success.", sagaAddrBase58)
		}
	}

	txHash, err := layer2Sdk.NeoVM.InvokeNeoVMContract(uint64(cfg.GasPrice), 200000, nil, cfg.AdminAccount, contractAddr, []interface{}{"StoreHash", []interface{}{"6de9439834c9147569741d3c9c9fc010"}})
	if err != nil {
		return fmt.Errorf("StoreUsedNum test failed %s", err)
	}

	_, err = common2.GetLayer2EventByTxHash(txHash.ToHexString(), cfg)
	if err != nil {
		return fmt.Errorf("init layer2 owner err1: %s", err)
	}
	log.Infof("test StoreUsedNum success ")
	return nil
}
