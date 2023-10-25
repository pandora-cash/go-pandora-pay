package api_common

import (
	"errors"
	"net/http"
	"pandora-pay/wallet"
)

type APIWalletImportMnemonicRequest struct {
	Mnemonic string `json:"mnemonic" msgpack:"mnemonic"`
}

type APIWalletImportMnemonicReply struct {
	Result bool `json:"result" msgpack:"mnemonic"`
}

func (api *APICommon) ImportWalletMnemonic(r *http.Request, args *APIWalletImportMnemonicRequest, reply *APIWalletImportMnemonicReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	if err = wallet.Wallet.ImportMnemonic(args.Mnemonic); err != nil {
		return
	}
	reply.Result = true

	return
}
