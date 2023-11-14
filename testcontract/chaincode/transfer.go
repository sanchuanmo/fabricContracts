package chaincode

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

/*
1.1版本 Asset的增删改查，范围查询，历史溯源
1.2版本 增加Asset的索引结构，索引查询
*/

const colorIdIndex string = "color~ID"
const assetGlobalName string = "assetGlobal"

type SmartContract struct {
	contractapi.Contract
}

type Asset struct {
	DocType        string `json:"docType"` //docType is used to distinguish the various types of objects in state database
	ID             string `json:"ID"`      //the field tags are needed to keep case from bouncing around
	Color          string `json:"color"`
	Size           int    `json:"size"`
	Owner          string `json:"owner"`
	AppraisedValue int    `json:"appraisedValue"`
}

type AssetGlobal struct {
	IdNum int `json:"idNum"`
}

type HistoryQueryResult struct {
	Record    *Asset    `json:"record"`
	TxId      string    `json:"txId"`
	Timestamp time.Time `json:"timestamp"`
	IsDelete  bool      `json:"isDelete"`
}

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	key := "assetGlobal"
	exist, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("can not find assetGlobal record, error :%v", err)
	}
	if exist != nil {
		return fmt.Errorf("assetGlobal exist, can not initLedger again.")
	}

	assetGlobalInstance := &AssetGlobal{
		IdNum: 0,
	}
	assetGlobalInstanceBytes, err := json.Marshal(assetGlobalInstance)
	if err != nil {
		return fmt.Errorf("json assetGlobalInstance error :%v", err)
	}

	err = ctx.GetStub().PutState(assetGlobalName, assetGlobalInstanceBytes)
	if err != nil {
		return fmt.Errorf("PutState assetGlobalInstance error:  %v", err)
	}
	return nil
}

func (s *SmartContract) ReadAssetGlobal(ctx contractapi.TransactionContextInterface) (*AssetGlobal, error) {
	assetGlobalBytes, err := ctx.GetStub().GetState(assetGlobalName)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset %s: %v", assetGlobalName, err)
	}
	if assetGlobalBytes == nil {
		return nil, fmt.Errorf("asset %s does not exist", assetGlobalName)
	}

	var assetGlobalInstance AssetGlobal
	err = json.Unmarshal(assetGlobalBytes, &assetGlobalInstance)
	if err != nil {
		return nil, err
	}

	return &assetGlobalInstance, nil
}

func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, assetID int) (*Asset, error) {
	assetBytes, err := ctx.GetStub().GetState(transIDToStr(assetID))
	if err != nil {
		return nil, fmt.Errorf("failed to get asset %d: %v", assetID, err)
	}
	if assetBytes == nil {
		return nil, fmt.Errorf("asset %d does not exist", assetID)
	}

	var asset Asset
	err = json.Unmarshal(assetBytes, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

func (s *SmartContract) AssetExists(ctx contractapi.TransactionContextInterface, assetID string) (bool, error) {
	assetBytes, err := ctx.GetStub().GetState(assetID)
	if err != nil {
		return false, fmt.Errorf("failed to read asset %s from world state. %v", assetID, err)
	}

	return assetBytes != nil, nil
}

func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, color string, size int, owner string, appraisedValue int) error {
	assetGlobalInstance, err := s.ReadAssetGlobal(ctx)
	if err != nil {
		return fmt.Errorf("get assetGlobalInstance error: %s", err)
	}

	var nextAssetID = assetGlobalInstance.IdNum + 1

	assetGlobalNewInstance := &AssetGlobal{
		IdNum: nextAssetID,
	}
	// id补零
	nextAssetIDStr := transIDToStr(nextAssetID)

	exists, err := s.AssetExists(ctx, nextAssetIDStr)
	if err != nil {
		return fmt.Errorf("failed to get asset: %v", err)
	}
	if exists {
		return fmt.Errorf("asset already exists: %s", nextAssetIDStr)
	}

	asset := &Asset{
		DocType:        "asset",
		ID:             nextAssetIDStr,
		Color:          color,
		Size:           size,
		Owner:          owner,
		AppraisedValue: appraisedValue,
	}

	assetBytes, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(nextAssetIDStr, assetBytes)
	if err != nil {
		return err
	}

	// 创建索引，基于Color进行范围查询，比如，范围所有颜色为blue 的资产Asset。
	// 该索引是个复合key，首先要列出对其进行范围查询的key。
	// 在本案例中，复合key是基于color~ID来构造的
	colorIdIndexKey, err := ctx.GetStub().CreateCompositeKey(colorIdIndex, []string{asset.Color, asset.ID})
	if err != nil {
		return err
	}
	value := []byte{0x00}
	err = ctx.GetStub().PutState(colorIdIndexKey, value)
	if err != nil {
		return fmt.Errorf("colorIdIndexKey put state error: %s", err)
	}

	assetGlobalNewInstanceByte, err := json.Marshal(assetGlobalNewInstance)

	if err != nil {
		return fmt.Errorf("assetGlobalNewInstance json marshal error:%s", err)
	}

	err = ctx.GetStub().PutState(assetGlobalName, assetGlobalNewInstanceByte)

	if err != nil {
		return fmt.Errorf("put assetGlobalName error:%s", err)
	}

	return nil
}

// 更新索引后，在updateAsset时也会更新对应索引字段
func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, assetID int, color string, size int, owner string, appraisedValue int) error {
	exists, err := s.AssetExists(ctx, transIDToStr(assetID))
	if err != nil {
		return fmt.Errorf("failed to get asset: %v", err)
	}
	if !exists {
		return fmt.Errorf("asset not exists: %d", assetID)
	}
	asset, err := s.ReadAsset(ctx, assetID)
	if err != nil {
		return err
	}
	err = updateIndex(ctx, colorIdIndex, []string{asset.Color, asset.ID}, []string{color, asset.ID})
	if err != nil {
		return fmt.Errorf("update asset index error:%s", err)
	}

	asset.Color = color
	asset.Size = size
	asset.Owner = owner
	asset.AppraisedValue = appraisedValue

	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return fmt.Errorf("asset json marshal error:%s", err)
	}

	return ctx.GetStub().PutState(transIDToStr(assetID), assetJSON)
}

func (s *SmartContract) DeleteAsset(ctx contractapi.TransactionContextInterface, assetID int) error {
	asset, err := s.ReadAsset(ctx, assetID)
	if err != nil {
		return fmt.Errorf("failed to get asset: %v", err)
	}
	if asset != nil {
		return fmt.Errorf("asset not exists: %d", assetID)
	}

	err = ctx.GetStub().DelState(transIDToStr(assetID))
	if err != nil {
		return fmt.Errorf("failed to delete asset %d: %v", assetID, err)
	}
	//删除记录的对应索引
	return delIndex(ctx, colorIdIndex, []string{asset.Color, asset.ID})
}

func (s *SmartContract) GetAssetsByRangeLatest(ctx contractapi.TransactionContextInterface, pageSize, pageIndex int) ([]*Asset, error) {
	assetGlobalInstance, err := s.ReadAssetGlobal(ctx)
	if err != nil {
		return nil, fmt.Errorf("get assetGlobalInstance error: %s", err)
	}
	latestKey := assetGlobalInstance.IdNum
	var endKey int = latestKey - (pageIndex-1)*pageSize + 1
	var startKey int = endKey - pageSize
	startKeyStr := transIDToStr(startKey)
	endKeyStr := transIDToStr(endKey)
	resultsIterator, _, err := ctx.GetStub().GetStateByRangeWithPagination(startKeyStr, endKeyStr, int32(pageSize), "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	return constructQueryResponseFromIterator(resultsIterator)
}

func (s *SmartContract) GetAssetHistory(ctx contractapi.TransactionContextInterface, assetID, pageSize, pageIndex int) ([]HistoryQueryResult, error) {
	log.Printf("GetAssetHistory: ID %v", assetID)

	resultsIterator, err := ctx.GetStub().GetHistoryForKey(transIDToStr(assetID))
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var records []HistoryQueryResult
	var index int = 0
	var startIndex int = (pageIndex - 1) * pageSize
	var endIndex int = startIndex + pageIndex

	for resultsIterator.HasNext() {
		if index < startIndex {
			_, err := resultsIterator.Next()
			if err != nil {
				return nil, err
			}
			index++
			continue
		}
		if index >= endIndex {
			break
		}
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var asset Asset
		if len(response.Value) > 0 {
			err = json.Unmarshal(response.Value, &asset)
			if err != nil {
				return nil, err
			}
		} else {
			asset = Asset{
				ID: transIDToStr(assetID),
			}
		}
		timestamp := response.Timestamp.AsTime()
		if err != nil {
			return nil, err
		}
		record := HistoryQueryResult{
			TxId:      response.TxId,
			Timestamp: timestamp,
			Record:    &asset,
			IsDelete:  response.IsDelete,
		}
		records = append(records, record)
		index++
	}
	return records, nil
}

// 富查询无需使用索引，但使用索引的查询会更有效率

// 常规使用索引富查询案例
func (s *SmartContract) QueryAssetsByColorIndex(ctx contractapi.TransactionContextInterface, color string) ([]*Asset, error) {
	coloredAssetResultsIterator, err := ctx.GetStub().GetStateByPartialCompositeKey(colorIdIndex, []string{color})
	if err != nil {
		return nil, err
	}
	defer coloredAssetResultsIterator.Close()

	var assets []*Asset
	for coloredAssetResultsIterator.HasNext() {
		responseRange, err := coloredAssetResultsIterator.Next()
		if err != nil {
			return nil, err
		}
		_, compositKeyParts, err := ctx.GetStub().SplitCompositeKey(responseRange.Key)
		if err != nil {
			return nil, err
		}
		if len(compositKeyParts) > 1 {
			returnedAssetID := compositKeyParts[1]
			returnedAssetIDInteger, err := strconv.Atoi(returnedAssetID)
			if err != nil {
				return nil, err
			}
			asset, err := s.ReadAsset(ctx, returnedAssetIDInteger)
			if err != nil {
				return nil, err
			}
			assets = append(assets, asset)
		}
	}
	return assets, nil
}

// 常规无索引富查询Color
func (s *SmartContract) QueryAssetsByColor(ctx contractapi.TransactionContextInterface, color string) ([]*Asset, error) {
	queryString := fmt.Sprintf(`{"selector":{"docType":"asset","color":"%s"}}`, color)
	return getQueryResultForQueryString(ctx, queryString)
}

// 常规无索引富查询案例
func (s *SmartContract) QueryAssets(ctx contractapi.TransactionContextInterface, queryString string) ([]*Asset, error) {
	return getQueryResultForQueryString(ctx, queryString)
}

type PaginatedQueryResult struct {
	Records             []*Asset `json:"records"`
	FetchedRecordsCount int32    `json:"fetchedRecordsCount"`
	Bookmark            string   `json:"bookmark"`
}

// 常规无索引富查询分页案例
func (s *SmartContract) QueryAssetsWithPagination(ctx contractapi.TransactionContextInterface, queryString string, pageSize int32, bookmark string) (*PaginatedQueryResult, error) {
	resultsIterator, responseMetadata, err := ctx.GetStub().GetQueryResultWithPagination(queryString, pageSize, bookmark)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	assets, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return nil, err
	}

	return &PaginatedQueryResult{
		Records:             assets,
		FetchedRecordsCount: responseMetadata.FetchedRecordsCount,
		Bookmark:            responseMetadata.Bookmark,
	}, nil
}

// 交易
func (s *SmartContract) TransferAssetByColor(ctx contractapi.TransactionContextInterface, color, newOwner string) error {
	coloredAssetResultsIterator, err := ctx.GetStub().GetStateByPartialCompositeKey(colorIdIndex, []string{color})
	if err != nil {
		return err
	}
	defer coloredAssetResultsIterator.Close()

	for coloredAssetResultsIterator.HasNext() {
		responseRange, err := coloredAssetResultsIterator.Next()
		if err != nil {
			return err
		}

		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(responseRange.Key)
		if err != nil {
			return err
		}

		if len(compositeKeyParts) > 1 {
			returnedAssetID := compositeKeyParts[1]
			returnedAssetIDInteger, err := strconv.Atoi(returnedAssetID)
			if err != nil {
				return err
			}
			asset, err := s.ReadAsset(ctx, returnedAssetIDInteger)
			if err != nil {
				return err
			}
			asset.Owner = newOwner
			assetBytes, err := json.Marshal(asset)
			if err != nil {
				return err
			}
			err = ctx.GetStub().PutState(returnedAssetID, assetBytes)
			if err != nil {
				return fmt.Errorf("transfer failed for asset %s: %v", returnedAssetID, err)
			}
		}
	}

	return nil

}
