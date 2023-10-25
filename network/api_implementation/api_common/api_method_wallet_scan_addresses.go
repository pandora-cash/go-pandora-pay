package api_common

import (
	"errors"
	"net/http"
	"pandora-pay/wallet"
)

type APIWalletScanAddressesReply struct {
	Result bool `json:"result" msgpack:"result"`
}

func (api *APICommon) GetWalletScanAddresses(r *http.Request, args *struct{}, reply *APIWalletScanAddressesReply, authenticated bool) (err error) {

	if !authenticated {
		return errors.New("Invalid User or Password")
	}

	if err = wallet.Wallet.ScanAddresses(); err != nil {
		return
	}
	reply.Result = true

	return
}
