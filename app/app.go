package app

import (
	"pandora-pay/address_balance_decryptor"
	"pandora-pay/blockchain"
	"pandora-pay/blockchain/forging"
	"pandora-pay/gui"
	"pandora-pay/mempool"
	"pandora-pay/network"
	"pandora-pay/settings"
	"pandora-pay/store"
	"pandora-pay/testnet"
	"pandora-pay/txs_builder"
	"pandora-pay/txs_validator"
	"pandora-pay/wallet"
)

var (
	Settings                *settings.Settings
	Wallet                  *wallet.Wallet
	Forging                 *forging.Forging
	Mempool                 *mempool.Mempool
	TxsValidator            *txs_validator.TxsValidator
	AddressBalanceDecryptor *address_balance_decryptor.AddressBalanceDecryptor
	Chain                   *blockchain.Blockchain
	Network                 *network.Network
	TxsBuilder              *txs_builder.TxsBuilder
	Testnet                 *testnet.Testnet
)

func Close() {
	store.DBClose()
	gui.GUI.Close()
	Forging.Close()
	Chain.Close()
	Wallet.Close()
}
