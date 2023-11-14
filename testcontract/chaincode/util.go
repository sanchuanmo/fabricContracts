package chaincode

import (
	"encoding/json"
	"strconv"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const idStringLong int = 16

func transIDToStr(assetID int) string {
	assetIDStr := strconv.Itoa(assetID)
	if len(assetIDStr) < idStringLong {
		size := idStringLong - len(assetIDStr)
		for i := 0; i < size; i++ {
			assetIDStr = "0" + assetIDStr
		}
	}
	return assetIDStr
}

func constructQueryResponseFromIterator(resultIterator shim.StateQueryIteratorInterface) ([]*Asset, error) {
	var assets []*Asset
	for resultIterator.HasNext() {
		queryresult, err := resultIterator.Next()
		if err != nil {
			return nil, err
		}
		var asset Asset
		err = json.Unmarshal(queryresult.Value, &asset)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}
	return assets, nil
}

func getQueryResultForQueryString(ctx contractapi.TransactionContextInterface, queryString string) ([]*Asset, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	return constructQueryResponseFromIterator(resultsIterator)
}

// 索引工具类

// 旧索引的Key，及构造新索引的参数
func updateIndex(ctx contractapi.TransactionContextInterface, indexName string, indexOldKeyValue, indexNewKeyValue []string) error {
	indexOldKey, err := ctx.GetStub().CreateCompositeKey(indexName, indexOldKeyValue)
	if err != nil {
		return err
	}
	indexNewKey, err := ctx.GetStub().CreateCompositeKey(indexName, indexNewKeyValue)
	if err != nil {
		return err
	}
	value := []byte{0x00}
	//写入新index
	err = ctx.GetStub().PutState(indexNewKey, value)
	if err != nil {
		return err
	}
	//删除旧index
	err = ctx.GetStub().DelState(indexOldKey)
	if err != nil {
		return err
	}
	return nil
}

func delIndex(ctx contractapi.TransactionContextInterface, indexName string, indexKeyValue []string) error {
	indexKey, err := ctx.GetStub().CreateCompositeKey(indexName, indexKeyValue)
	if err != nil {
		return err
	}
	return ctx.GetStub().DelState(indexKey)
}
