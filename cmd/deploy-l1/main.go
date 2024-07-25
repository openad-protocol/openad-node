package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/howeyc/gopass"
	ontology_go_sdk "github.com/ontio/ontology-go-sdk"
	"github.com/ontio/ontology/common"
	"github.com/urfave/cli/v2"
)

const ()

func rootCmd(c *cli.Context) error {
	cfwlt := c.String("wallet")
	cfRest := c.String("resturl")
	log.Printf("rading wallet: %s, send request to: %s", cfwlt, cfRest)
	sdk := ontology_go_sdk.NewOntologySdk()
	sdk.NewRpcClient().SetAddress(cfRest)

	wall, err := ontology_go_sdk.OpenWallet(cfwlt)
	if err != nil {
		return err
	}
	fmt.Println("input wallet passwd:")
	psw, err := gopass.GetPasswd()
	if err != nil {
		return err
	}
	act, err := wall.GetDefaultAccount(psw)
	if err != nil {
		return err
	}
	log.Printf("get default account OK, using account: %s", act.Address.ToBase58())
	b, err := ioutil.ReadFile(c.String("contract"))
	if err != nil {
		return err
	}
	contractHex := strings.Trim(string(b), "\n ")

	contract, err := hex.DecodeString(contractHex)
	if err != nil {
		return err
	}

	addr := common.AddressFromVmCode(contract)
	log.Printf("contract address: %s\n", addr.ToHexString())

	// 	                                                                                     name,  version, author, email, desc
	txHash, err := sdk.NeoVM.DeployNeoVMSmartContract(2500, 21000000, act, true, contractHex, "apmainnet", "0.1", "apllc", "techstaff@apllc.jp", "zapc mainnet contract")
	if err != nil {
		return err
	}
	log.Printf("%s", txHash.ToHexString())

	time.Sleep(10 * time.Second)

	empty, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		return err
	}

	// res, err := sdk.NeoVM.PreExecInvokeNeoVMContract(addr, []interface{}{"init", []interface{}{act.Address, []interface{}{empty, 0, "1"}, 2}})
	res, err := sdk.NeoVM.InvokeNeoVMContract(2500, 21000000, act, act, addr, []interface{}{"init", []interface{}{act.Address, []interface{}{empty, 0, "1"}, 2}})

	if err != nil {
		return err
	}
	log.Println(res.ToHexString())

	return nil
}

func main() {
	app := &cli.App{
		Name: "deploy",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "resturl",
				Value: "http://polaris3.ont.io:20336",
				Usage: "the remote url, default to testnet",
			},

			&cli.StringFlag{
				Name:  "wallet",
				Value: "wallet.dat",
				Usage: "the wallet to deploy this contract",
			},
			&cli.StringFlag{
				Name:  "contract",
				Value: "a.contract",
				Usage: "the contract binary hex to deploy this contract",
			},
		},
		Usage:  "deploy l2 contract",
		Action: rootCmd,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
