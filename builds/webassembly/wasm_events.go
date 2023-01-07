package main

import (
	"encoding/base64"
	"errors"
	"pandora-pay/blockchain/data_storage/accounts/account"
	"pandora-pay/blockchain/data_storage/assets/asset"
	"pandora-pay/blockchain/data_storage/plain_accounts/plain_account"
	"pandora-pay/blockchain/data_storage/registrations/registration"
	"pandora-pay/builds/webassembly/webassembly_utils"
	"pandora-pay/config/globals"
	"pandora-pay/helpers/advanced_buffers"
	"pandora-pay/helpers/msgpack"
	"pandora-pay/helpers/recovery"
	"pandora-pay/network/api_code/api_code_types"
	"pandora-pay/network/api_code/api_code_websockets"
	"pandora-pay/network/api_implementation/api_common/api_types"
	"sync/atomic"
	"syscall/js"
)

func listenEvents(this js.Value, args []js.Value) interface{} {

	if len(args) == 0 || args[0].Type() != js.TypeFunction {
		return errors.New("Argument must be a callback")
	}

	index := atomic.AddUint64(&subscriptionsIndex, 1)
	channel := globals.MainEvents.AddListener()

	callback := args[0]
	var err error

	recovery.SafeGo(func() {
		for {
			data, ok := <-channel
			if !ok {
				return
			}

			var final interface{}

			switch v := data.Data.(type) {
			case string:
				final = data.Data
			case interface{}:
				if final, err = webassembly_utils.ConvertJSONBytes(v); err != nil {
					panic(err)
				}
			default:
				final = data.Data
			}

			callback.Invoke(data.Name, final)
		}
	})

	return index
}

func listenNetworkNotifications(this js.Value, args []js.Value) interface{} {
	return webassembly_utils.PromiseFunction(func() (interface{}, error) {

		if len(args) != 1 || args[0].Type() != js.TypeFunction {
			return nil, errors.New("Argument must be a callback function")
		}
		callback := args[0]

		subscriptionsCn := api_code_websockets.SubscriptionNotifications.AddListener()

		recovery.SafeGo(func() {

			defer api_code_websockets.SubscriptionNotifications.RemoveChannel(subscriptionsCn)

			var err error
			for {
				data, ok := <-subscriptionsCn
				if !ok {
					return
				}

				func() {

					var object, extra any

					//gui.GUI.Log(int(data.SubscriptionType))

					switch data.SubscriptionType {
					case api_code_types.SUBSCRIPTION_ACCOUNT:
						var acc *account.Account
						if data.Data != nil {

							if acc, err = account.NewAccount(data.Key, 0, nil); err != nil {
								return
							}
							if err = acc.Deserialize(advanced_buffers.NewBufferReader(data.Data)); err != nil {
								return
							}
						}
						object = acc
						extra = &api_types.APISubscriptionNotificationAccountExtra{}
					case api_code_types.SUBSCRIPTION_PLAIN_ACCOUNT:
						plainAcc := plain_account.NewPlainAccount(data.Key, 0)
						if data.Data != nil {
							if err = plainAcc.Deserialize(advanced_buffers.NewBufferReader(data.Data)); err != nil {
								return
							}
						}
						object = plainAcc
						extra = &api_types.APISubscriptionNotificationPlainAccExtra{}
					case api_code_types.SUBSCRIPTION_ASSET:
						ast := asset.NewAsset(data.Key, 0)
						if data.Data != nil {
							if err = ast.Deserialize(advanced_buffers.NewBufferReader(data.Data)); err != nil {
								return
							}
						}
						object = ast

						extra = &api_types.APISubscriptionNotificationAssetExtra{}
					case api_code_types.SUBSCRIPTION_REGISTRATION:
						reg := registration.NewRegistration(data.Key, 0)
						if data.Data != nil {
							if err = reg.Deserialize(advanced_buffers.NewBufferReader(data.Data)); err != nil {
								return
							}
						}
						object = reg
						extra = &api_types.APISubscriptionNotificationRegistrationExtra{}
					case api_code_types.SUBSCRIPTION_ACCOUNT_TRANSACTIONS:
						object = data.Data
						extra = &api_types.APISubscriptionNotificationAccountTxExtra{}
					case api_code_types.SUBSCRIPTION_TRANSACTION:
						object = data.Data
						extra = &api_types.APISubscriptionNotificationTxExtra{}
					default:
						return //invalid
					}

					if data.Extra != nil {
						if err = msgpack.Unmarshal(data.Extra, extra); err != nil {
							return
						}
					} else {
						extra = nil
					}

					jsOutData, err1 := webassembly_utils.ConvertJSONBytes(object)
					jsOutExtra, err2 := webassembly_utils.ConvertJSONBytes(extra)

					if err1 != nil || err2 != nil {
						return
					}
					callback.Invoke(int(data.SubscriptionType), base64.StdEncoding.EncodeToString(data.Key), jsOutData, jsOutExtra)

				}()

			}
		})

		return true, nil
	})
}
