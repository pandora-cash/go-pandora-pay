package api_common

import (
	"errors"
	"net/http"
	"pandora-pay/network/api_implementation/api_common/api_types"
	"pandora-pay/wallet"
	"pandora-pay/wallet/wallet_address"
)

type APIWalletGetAddressRequest struct {
	api_types.APIAccountBaseRequest
	Index int `json:"index" msgpack:"index"`
}

type APIWalletGetAddressReply struct {
	Address *wallet_address.WalletAddress `json:"addresses" msgpack:"addresses"`
}

func (api *APICommon) GetWalletAddress(r *http.Request, args *APIWalletGetAddressRequest, reply *APIWalletGetAddressReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	if args.Index >= 0 {
		if reply.Address, err = wallet.Wallet.GetWalletAddress(args.Index, true); err != nil {
			return
		}
	} else {
		var publicKey []byte
		if publicKey, err = args.GetPublicKey(true); err != nil {
			return
		}
		reply.Address = wallet.Wallet.GetWalletAddressByPublicKey(publicKey, true)
	}

	return
}
