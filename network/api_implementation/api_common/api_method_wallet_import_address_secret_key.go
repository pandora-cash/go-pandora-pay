package api_common

import (
	"errors"
	"net/http"
	"pandora-pay/helpers"
	"pandora-pay/wallet"
	"pandora-pay/wallet/wallet_address"
)

type APIWalletImportAddressSecretKeyRequest struct {
	Name          string         `json:"name" msgpack:"name""`
	SecretKey     helpers.Base64 `json:"mnemonic" msgpack:"mnemonic"`
	Staked        bool           `json:"staked" msgpack:"staked"`
	SpendRequired bool           `json:"spendRequired" msgpack:"spendRequired"`
}

type APIWalletImportAddressSecretKeyReply struct {
	Address *wallet_address.WalletAddress `json:"address" msgpack:"address"`
	Result  bool                          `json:"result" msgpack:"result"`
}

func (api *APICommon) ImportWalletAddressSecretKey(r *http.Request, args *APIWalletImportAddressSecretKeyRequest, reply *APIWalletImportAddressSecretKeyReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	var addr *wallet_address.WalletAddress
	if addr, err = wallet.Wallet.ImportSecretKey(args.Name, args.SecretKey, args.Staked, args.SpendRequired); err != nil {
		return err
	}
	reply.Address = addr
	reply.Result = true

	return
}
