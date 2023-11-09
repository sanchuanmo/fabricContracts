package chaincode

import (
	"encoding/json"
	"strconv"

	"github.com/hyperledger/fabric-chaincode-go/shim"
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
