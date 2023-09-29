package blockchain

import (
	"errors"
	"math/big"
	"pandora-pay/addresses"
	"pandora-pay/blockchain/blockchain_types"
	"pandora-pay/blockchain/blocks/block"
	"pandora-pay/blockchain/blocks/block_complete"
	"pandora-pay/blockchain/data_storage"
	"pandora-pay/blockchain/data_storage/accounts"
	"pandora-pay/blockchain/data_storage/accounts/account"
	"pandora-pay/blockchain/data_storage/assets/asset"
	"pandora-pay/blockchain/data_storage/registrations"
	"pandora-pay/blockchain/forging/forging_block_work"
	"pandora-pay/blockchain/genesis"
	"pandora-pay/blockchain/transactions/transaction"
	"pandora-pay/config"
	"pandora-pay/config/config_coins"
	"pandora-pay/config/config_forging"
	"pandora-pay/config/config_stake"
	"pandora-pay/cryptography"
	"pandora-pay/gui"
	"pandora-pay/helpers"
	"pandora-pay/helpers/advanced_buffers"
	"pandora-pay/helpers/recovery"
	"pandora-pay/mempool"
	"pandora-pay/network/websocks/connection/advanced_connection_types"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
)

func (self *blockchain) GetChainData() *BlockchainData {
	return self.ChainData.Load()
}

func (self *blockchain) GetChainDataUpdate() *BlockchainDataUpdate {
	return &BlockchainDataUpdate{self.ChainData.Load(), self.Sync.GetSyncData()}
}

func (self *blockchain) createGenesisBlockchainData() *BlockchainData {
	return &BlockchainData{
		helpers.CloneBytes(genesis.GenesisData.Hash),
		helpers.CloneBytes(genesis.GenesisData.Hash),
		helpers.CloneBytes(genesis.GenesisData.KernelHash),
		helpers.CloneBytes(genesis.GenesisData.KernelHash),
		0,
		0,
		new(big.Int).SetBytes(helpers.CloneBytes(genesis.GenesisData.Target)),
		new(big.Int).SetUint64(0),
		0,
		0,
		0,
		0,
		0,
	}
}

func (self *blockchain) initializeNewChain(chainData *BlockchainData, dataStorage *data_storage.DataStorage) (err error) {

	gui.GUI.Info("Initializing New Chain")

	supply := uint64(0)

	for _, airdrop := range genesis.GenesisData.AirDrops {

		if err = helpers.SafeUint64Add(&supply, airdrop.Amount); err != nil {
			return
		}

		var addr *addresses.Address
		addr, err = addresses.DecodeAddr(airdrop.Address)
		if err != nil {
			return
		}
		if addr.IsIntegratedAmount() || addr.IsIntegratedPaymentID() || addr.IsIntegratedPaymentAsset() {
			return errors.New("Amount, PaymentID or IntegratedPaymentAsset are not allowed in the airdrop address")
		}

		if registrations.VerifyRegistration(addr.PublicKey, addr.Staked, addr.SpendPublicKey, addr.Registration) == false {
			return errors.New("Registration verification is false")
		}

		if _, err = dataStorage.CreateRegistration(addr.PublicKey, addr.Staked, addr.SpendPublicKey); err != nil {
			return
		}

		var accs *accounts.Accounts
		var acc *account.Account

		if accs, acc, err = dataStorage.CreateAccount(config_coins.NATIVE_ASSET_FULL, addr.PublicKey, false); err != nil {
			return
		}
		acc.Balance.AddBalanceUint(airdrop.Amount)

		if err = accs.Update(string(addr.PublicKey), acc); err != nil {
			return
		}

	}

	ast := &asset.Asset{
		nil,
		0,
		0,
		false,
		false,
		false,
		false,
		false,
		false,
		false,
		byte(config_coins.DECIMAL_SEPARATOR),
		config_coins.MAX_SUPPLY_COINS_UNITS,
		supply,
		config_coins.BURN_PUBLIC_KEY,
		config_coins.BURN_PUBLIC_KEY,
		config_coins.NATIVE_ASSET_NAME,
		config_coins.NATIVE_ASSET_TICKER,
		config_coins.NATIVE_ASSET_IDENTIFICATION,
		config_coins.NATIVE_ASSET_DESCRIPTION,
		nil,
	}

	if err = dataStorage.Asts.CreateAsset(config_coins.NATIVE_ASSET_FULL, ast); err != nil {
		return
	}

	if err = dataStorage.CommitChanges(); err != nil {
		return
	}

	chainData.AssetsCount = dataStorage.Asts.Count
	chainData.AccountsCount = dataStorage.Regs.Count + dataStorage.PlainAccs.Count

	return
}

func (self *blockchain) init() (*BlockchainData, error) {

	chainData := self.createGenesisBlockchainData()

	if err := store.StoreBlockchain.DB.Update(func(writer store_db_interface.StoreDBTransactionInterface) (err error) {

		dataStorage := data_storage.NewDataStorage(writer)

		if config.NODE_CONSENSUS == config.NODE_CONSENSUS_TYPE_FULL {
			if err = self.initializeNewChain(chainData, dataStorage); err != nil {
				return
			}
		}

		if config.NODE_PROVIDE_EXTENDED_INFO_APP {
			if err = saveAssetsInfo(dataStorage.Asts); err != nil {
				return
			}
		}

		return

	}); err != nil {
		return nil, err
	}

	self.ChainData.Store(chainData)
	return chainData, nil
}

func (self *blockchain) createNextBlockForForging(chainData *BlockchainData, newWork bool) {

	if config.NODE_CONSENSUS != config.NODE_CONSENSUS_TYPE_FULL {
		return
	}

	if chainData == nil {
		mempool.Mempool.ContinueWork()
	} else {
		mempool.Mempool.UpdateWork(chainData.Hash, chainData.Height)
	}

	if !config_forging.FORGING_ENABLED {
		return
	}

	if newWork {

		if chainData == nil {
			chainData = self.GetChainData()
		}

		target := chainData.Target

		var blk *block.Block
		var err error
		if chainData.Height == 0 {
			if blk, err = genesis.CreateNewGenesisBlock(); err != nil {
				gui.GUI.Error("Error creating next block", err)
				return
			}
		} else {
			blk = &block.Block{
				BlockHeader: &block.BlockHeader{
					Version: 0,
					Height:  chainData.Height,
				},
				MerkleHash:     cryptography.SHA3([]byte{}),
				PrevHash:       chainData.Hash,
				PrevKernelHash: chainData.KernelHash,
				Timestamp:      chainData.Timestamp,
			}
		}

		blk.StakingNonce = make([]byte, 32)

		blk.BloomSerializedNow(blk.SerializeManualToBytes())

		blkComplete := &block_complete.BlockComplete{
			Block: blk,
			Txs:   []*transaction.Transaction{},
		}

		if err = blkComplete.BloomCompleteBySerialized(blkComplete.SerializeManualToBytes()); err != nil {
			return
		}

		writer := advanced_buffers.NewBufferWriter()
		blk.SerializeForForging(writer)

		self.NextBlockCreatedCn <- &forging_block_work.ForgingWork{
			blkComplete,
			writer.Bytes(),
			blkComplete.Timestamp,
			blkComplete.Height,
			target,
			config_stake.GetRequiredStake(blkComplete.Height),
		}

	}

}

func (self *blockchain) InitForging() {

	recovery.SafeGo(func() {

		for {

			solution, ok := <-self.ForgingSolutionCn
			if !ok {
				return
			}

			kernelHash, err := self.AddBlocks([]*block_complete.BlockComplete{solution.BlkComplete}, true, advanced_connection_types.UUID_ALL)

			solution.Done <- &blockchain_types.BlockchainSolutionAnswer{
				err,
				kernelHash,
			}
		}

	})

	recovery.SafeGo(func() {

		updateNewSyncCn := self.Sync.UpdateSyncMulticast.AddListener()
		defer self.Sync.UpdateSyncMulticast.RemoveChannel(updateNewSyncCn)

		for {

			newSync := <-updateNewSyncCn

			if newSync.Started {
				self.createNextBlockForForging(self.GetChainData(), true)
				break
			}

		}
	})

}
