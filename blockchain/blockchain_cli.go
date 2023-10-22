package blockchain

import (
	"context"
	"pandora-pay/blockchain/blockchain_types"
	"pandora-pay/blockchain/blocks/block_complete"
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/config"
	"pandora-pay/gui"
	"pandora-pay/mempool"
	"pandora-pay/network/websocks/connection/advanced_connection_types"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
)

func (self *blockchain) CliNewBlockchainTop(cmd string, ctx context.Context) (err error) {

	c := self.GetChainData()

	newTop := gui.GUI.OutputReadUint64("New blockchain top ?", false, 0, func(v uint64) bool {
		return v < c.Height
	})

	self.mutex.Lock()
	defer self.mutex.Unlock()

	var newChainData *BlockchainData
	var dataStorage *data_storage.DataStorage
	removedTxsList := [][]byte{} //ordered list
	removedTxHashes := make(map[string][]byte)
	allTransactionsChanges := []*blockchain_types.BlockchainTransactionUpdate{}

	mempool.Mempool.SuspendProcessingCn <- struct{}{}

	err = store.StoreBlockchain.DB.Update(func(writer store_db_interface.StoreDBTransactionInterface) (err error) {

		defer func() {
			if errReturned := recover(); errReturned != nil {
				err = errReturned.(error)
			}
		}()

		chainData := self.GetChainData()
		newChainData = chainData.clone()

		dataStorage = data_storage.NewDataStorage(writer)

		var removedBlocksTransactionsCount uint64

		height := newChainData.Height
		for height > newTop {

			if allTransactionsChanges, err = self.removeBlockComplete(writer, height-1, removedTxHashes, allTransactionsChanges, dataStorage); err != nil {
				return
			}
			if err = self.deleteUnusedBlocksComplete(writer, height-1, dataStorage); err != nil {
				return
			}

			height--
		}

		if err = dataStorage.CommitChanges(); err != nil {
			return
		}

		removedBlocksTransactionsCount = newChainData.TransactionsCount

		if height == 0 {
			gui.GUI.Info("chain.createGenesisBlockchainData called")
			newChainData = self.createGenesisBlockchainData()
		} else {
			newChainData = &BlockchainData{}
			if err = newChainData.loadBlockchainInfo(writer, height); err != nil {
				return
			}
		}

		//removing unused transactions
		if config.NODE_PROVIDE_EXTENDED_INFO_APP {
			removeUnusedTransactions(writer, newChainData.TransactionsCount, removedBlocksTransactionsCount)
		}

		for _, change := range allTransactionsChanges {
			if removedTxHashes[change.TxHashStr] != nil {
				writer.Delete("tx:" + change.TxHashStr)
				writer.Delete("txHash:" + change.TxHashStr)
				writer.Delete("txBlock:" + change.TxHashStr)
			}
		}

		if config.NODE_PROVIDE_EXTENDED_INFO_APP {
			removeTxsInfo(writer, removedTxHashes)
		}

		if err = self.saveBlockchainHashmaps(dataStorage); err != nil {
			return
		}

		return
	})

	if err == nil && newChainData != nil {
		self.ChainData.Store(newChainData)
		mempool.Mempool.ContinueProcessingCn <- mempool.CONTINUE_PROCESSING_NO_ERROR
	} else {
		mempool.Mempool.ContinueProcessingCn <- mempool.CONTINUE_PROCESSING_ERROR
	}

	update := &blockchainUpdate{
		err:              err,
		calledByForging:  false,
		exceptSocketUUID: advanced_connection_types.UUID_ALL,
	}

	if err == nil && newChainData != nil {
		update.newChainData = newChainData
		update.dataStorage = dataStorage
		update.removedTxsList = removedTxsList
		update.removedTxHashes = removedTxHashes
		update.insertedTxs = make(map[string]*transaction.Transaction)
		update.insertedTxsList = make([]*transaction.Transaction, 0)
		update.insertedBlocks = []*block_complete.BlockComplete{}
		update.allTransactionsChanges = allTransactionsChanges
	}

	self.updatesQueue.updatesCn <- update

	return
}

func (self *blockchain) initBlockchainCLI() {
	gui.GUI.CommandDefineCallback("New Blockchain Top", self.CliNewBlockchainTop, true)
}