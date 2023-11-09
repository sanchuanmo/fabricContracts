package main

import (
	"log"
	"testcontract/chaincode"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	assetChaincode, err := contractapi.NewChaincode(&chaincode.SmartContract{})

	if err != nil {
		log.Panicf("Error creating test contract chaincode: %v", err)
	}
	if err := assetChaincode.Start(); err != nil {
		log.Panicf("Error starting test contract chaincode: %v", err)
	}
}
