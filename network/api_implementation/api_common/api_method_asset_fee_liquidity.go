package api_common

import (
	"errors"
	"net/http"
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/data_storage/plain_accounts/plain_account"
	"pandora-pay/blockchain/data_storage/plain_accounts/plain_account/asset_fee_liquidity"
	"pandora-pay/helpers"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
)

type APIAssetFeeLiquidityFeeRequest struct {
	Height uint64         `json:"height,omitempty" msgpack:"height,omitempty"`
	Hash   helpers.Base64 `json:"hash,omitempty" msgpack:"hash,omitempty"`
}

type APIAssetFeeLiquidityFeeReply struct {
	Asset        []byte `json:"asset" msgpack:"asset"`
	Rate         uint64 `json:"rate" msgpack:"rate"`
	LeadingZeros byte   `json:"leadingZeros" msgpack:"leadingZeros"`
	Collector    []byte `json:"collector"  msgpack:"collector"` //collector Public Key
}

func (api *APICommon) GetAssetFeeLiquidity(r *http.Request, args *APIAssetFeeLiquidityFeeRequest, reply *APIAssetFeeLiquidityFeeReply) error {
	return store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

		if args.Hash == nil {
			if args.Hash, err = api.ApiStore.loadAssetHash(reader, args.Height); err != nil {
				return
			}
		}

		dataStorage := data_storage.NewDataStorage(reader)

		var plainAcc *plain_account.PlainAccount
		if plainAcc, err = dataStorage.GetWhoHasAssetTopLiquidity(args.Hash); err != nil || plainAcc == nil {
			return helpers.ReturnErrorIfNot(err, "Error retrieving Who Has Asset TopLiqiduity")
		}

		var liquditity *asset_fee_liquidity.AssetFeeLiquidity
		if liquditity = plainAcc.AssetFeeLiquidities.GetLiquidity(args.Hash); liquditity == nil {
			return errors.New("Error. It should have the liquidity")
		}

		reply.Asset = args.Hash
		reply.Rate = liquditity.Rate
		reply.LeadingZeros = liquditity.LeadingZeros
		reply.Collector = plainAcc.AssetFeeLiquidities.Collector

		return
	})
}
