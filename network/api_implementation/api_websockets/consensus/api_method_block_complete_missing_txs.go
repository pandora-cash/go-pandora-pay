package consensus

import (
	"errors"
	"net/http"
	"pandora-pay/helpers"
	"pandora-pay/helpers/msgpack"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
	"strconv"
)

type APIBlockCompleteMissingTxsRequest struct {
	Hash       helpers.Base64 `json:"hash,omitempty" msgpack:"hash,omitempty"`
	MissingTxs []int          `json:"missingTxs,omitempty" msgpack:"missingTxs,omitempty"`
}

type APIBlockCompleteMissingTxsReply struct {
	Txs [][]byte `json:"txs,omitempty" msgpack:"txs,omitempty"`
}

func (api *Consensus) GetBlockCompleteMissingTxs(r *http.Request, args *APIBlockCompleteMissingTxsRequest, reply *APIBlockCompleteMissingTxsReply) error {
	return store.StoreBlockchain.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

		heightStr := reader.Get("blockHeight_ByHash" + string(args.Hash))
		if heightStr == nil {
			return errors.New("Block was not found by hash")
		}

		var height uint64
		if height, err = strconv.ParseUint(string(heightStr), 10, 64); err != nil {
			return
		}

		data := reader.Get("blockTxs" + strconv.FormatUint(height, 10))
		if data == nil {
			return errors.New("Block not found")
		}

		txHashes := [][]byte{}
		if err = msgpack.Unmarshal(data, &txHashes); err != nil {
			return
		}

		reply.Txs = make([][]byte, len(args.MissingTxs))
		for i, txMissingIndex := range args.MissingTxs {
			if txMissingIndex >= 0 && txMissingIndex < len(txHashes) {
				tx := reader.Get("tx:" + string(txHashes[txMissingIndex]))
				if tx == nil {
					return errors.New("Tx was not found")
				}
				reply.Txs[i] = tx
			}
		}

		return
	})
}
