package websocks

import (
	"encoding/json"
	"pandora-pay/blockchain"
	"pandora-pay/blockchain/accounts"
	"pandora-pay/blockchain/accounts/account"
	"pandora-pay/config"
	"pandora-pay/helpers"
	"pandora-pay/network/api/api-common/api_types"
	"pandora-pay/network/websocks/connection"
)

type WebsocketSubscriptions struct {
	websockets            *Websockets
	chain                 *blockchain.Blockchain
	websocketClosedCn     chan *connection.AdvancedConnection
	newSubscriptionCn     chan *connection.SubscriptionNotification
	removeSubscriptionCn  chan *connection.SubscriptionNotification
	accountsSubscriptions map[string]map[string]*connection.SubscriptionNotification
	tokensSubscriptions   map[string]map[string]*connection.SubscriptionNotification
}

func newWebsocketSubscriptions(websockets *Websockets, chain *blockchain.Blockchain) (subs *WebsocketSubscriptions) {
	subs = &WebsocketSubscriptions{
		websockets, chain, make(chan *connection.AdvancedConnection),
		make(chan *connection.SubscriptionNotification),
		make(chan *connection.SubscriptionNotification),
		make(map[string]map[string]*connection.SubscriptionNotification),
		make(map[string]map[string]*connection.SubscriptionNotification),
	}

	if config.SEED_WALLET_NODES_INFO {
		go subs.processSubscriptions()
	}

	return
}

func (subs *WebsocketSubscriptions) send(apiRoute []byte, key []byte, list map[string]*connection.SubscriptionNotification, data helpers.SerializableInterface) {

	var err error
	var bytes []byte
	var serialized, marshalled *api_types.APISubscriptionNotification

	for _, subNot := range list {

		if data == nil {
			subNot.Conn.Send(key, nil)
			continue
		}

		if subNot.Subscription.ReturnType == api_types.RETURN_SERIALIZED {
			if serialized == nil {
				serialized = &api_types.APISubscriptionNotification{key, data.SerializeToBytes()}
			}
			subNot.Conn.SendJSON(apiRoute, serialized)
		} else if subNot.Subscription.ReturnType == api_types.RETURN_JSON {
			if marshalled == nil {
				if bytes, err = json.Marshal(data); err != nil {
					panic(err)
				}
				marshalled = &api_types.APISubscriptionNotification{key, bytes}
			}
			subNot.Conn.SendJSON(apiRoute, marshalled)
		}

	}
}

func (subs *WebsocketSubscriptions) getSubsMap(subscriptionType api_types.SubscriptionType) (subsMap map[string]map[string]*connection.SubscriptionNotification) {
	switch subscriptionType {
	case api_types.SUBSCRIPTION_ACCOUNT:
		subsMap = subs.accountsSubscriptions
	}
	return
}

func (subs *WebsocketSubscriptions) processSubscriptions() {

	updateAccountsCn := subs.chain.UpdateAccounts.AddListener()

	var subsMap map[string]map[string]*connection.SubscriptionNotification

	for {

		select {
		case subscription, ok := <-subs.newSubscriptionCn:
			if !ok {
				return
			}

			if subsMap = subs.getSubsMap(subscription.Subscription.Type); subsMap == nil {
				continue
			}

			keyStr := string(subscription.Subscription.Key)
			if subsMap[keyStr] == nil {
				subsMap[keyStr] = make(map[string]*connection.SubscriptionNotification)
			}
			subsMap[keyStr][subscription.Conn.UUID] = subscription

		case subscription, ok := <-subs.removeSubscriptionCn:
			if !ok {
				return
			}

			if subsMap = subs.getSubsMap(subscription.Subscription.Type); subsMap == nil {
				continue
			}

			keyStr := string(subscription.Subscription.Key)
			if subsMap[keyStr] != nil {
				delete(subsMap[keyStr], subscription.Conn.UUID)
				if len(subsMap[keyStr]) == 0 {
					delete(subsMap, keyStr)
				}
			}

		case accsData, ok := <-updateAccountsCn:
			if !ok {
				return
			}

			accs := accsData.(*accounts.Accounts)
			for k, v := range accs.HashMap.Committed {
				list := subs.accountsSubscriptions[k]
				if list != nil {
					subs.send([]byte("sub/notify"), []byte(k), list, v.Element.(*account.Account))
				}
			}
		case conn, ok := <-subs.websocketClosedCn:
			if !ok {
				return
			}

			var deleted []string
			for key, value := range subs.accountsSubscriptions {
				if value[conn.UUID] != nil {
					delete(value, conn.UUID)
				}
				if len(value) == 0 {
					deleted = append(deleted, key)
				}
			}
			for _, key := range deleted {
				delete(subs.accountsSubscriptions, key)
			}

		}

	}

}