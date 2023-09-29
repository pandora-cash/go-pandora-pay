package chain_network

import (
	"pandora-pay/config"
	"pandora-pay/helpers/recovery"
	"pandora-pay/network/api_implementation/api_websockets/consensus"
	"pandora-pay/network/server/node_http"
	"pandora-pay/network/websocks"
	"pandora-pay/network/websocks/connection"
	"time"
)

func continuouslyDownloadChain() {
	recovery.SafeGo(func() {

		for {

			list := websocks.Websockets.GetAllSockets()
			for _, conn := range list {
				if conn.Handshake.Consensus == config.NODE_CONSENSUS_TYPE_FULL {
					data, err := connection.SendJSONAwaitAnswer[consensus.ChainUpdateNotification](conn, []byte("get-chain"), nil, nil, 0)
					if err == nil {
						node_http.HttpServer.ApiWebsockets.Consensus.ChainUpdateProcess(conn, data)
					}
					time.Sleep(1 * time.Millisecond)
				}
			}

			time.Sleep(2000 * time.Millisecond)
		}

	})
}

func continuouslyDownloadMempool() {

	recovery.SafeGo(func() {

		for {

			list := websocks.Websockets.GetAllSockets()
			for _, conn := range list {
				if config.NODE_CONSENSUS == config.NODE_CONSENSUS_TYPE_FULL && conn.Handshake.Consensus == config.NODE_CONSENSUS_TYPE_FULL {
					DownloadMempool(conn)
					time.Sleep(1 * time.Millisecond)
				}
			}

			time.Sleep(2000 * time.Millisecond)
		}

	})

}

func syncBlockchainNewConnections() {
	recovery.SafeGo(func() {

		cn := websocks.Websockets.UpdateNewConnectionMulticast.AddListener()
		defer websocks.Websockets.UpdateNewConnectionMulticast.RemoveChannel(cn)

		for {

			conn, ok := <-cn
			if !ok {
				return
			}

			//making it async
			recovery.SafeGo(func() {

				data, err := connection.SendJSONAwaitAnswer[consensus.ChainUpdateNotification](conn, []byte("get-chain"), nil, nil, 0)
				if err == nil {
					node_http.HttpServer.ApiWebsockets.Consensus.ChainUpdateProcess(conn, data)
				}

			})

		}
	})
}

func Initialize() {

	continuouslyDownloadChain()

	if config.NODE_CONSENSUS == config.NODE_CONSENSUS_TYPE_FULL {
		continuouslyDownloadMempool()
	}

	syncBlockchainNewConnections()

	initializeConsensus()

}
