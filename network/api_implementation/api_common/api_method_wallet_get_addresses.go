package api_common

import (
	"errors"
	"net/http"
	"pandora-pay/wallet"
	"pandora-pay/wallet/wallet_address"
)

type APIWalletGetAddressesReply struct {
	Addresses []*wallet_address.WalletAddress `json:"addresses" msgpack:"addresses"`
}

func (api *APICommon) GetWalletAddresses(r *http.Request, args *struct{}, reply *APIWalletGetAddressesReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	wallet.Wallet.Lock.RLock()
	defer wallet.Wallet.Lock.RUnlock()

	reply.Addresses = make([]*wallet_address.WalletAddress, len(wallet.Wallet.Addresses))
	for i, addr := range wallet.Wallet.Addresses {
		reply.Addresses[i] = addr.Clone()
	}

	return
}
