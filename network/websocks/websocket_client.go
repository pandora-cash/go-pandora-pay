package websocks

import (
	"context"
	"nhooyr.io/websocket"
	"pandora-pay/config"
	"pandora-pay/network/known_nodes/known_node"
	"pandora-pay/network/websocks/connection"
)

type WebsocketClient struct {
	knownNode  *known_node.KnownNodeScored
	conn       *connection.AdvancedConnection
	websockets *Websockets
}

func NewWebsocketClient(websockets *Websockets, knownNode *known_node.KnownNodeScored) (*WebsocketClient, error) {

	wsClient := &WebsocketClient{
		knownNode:  knownNode,
		websockets: websockets,
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.WEBSOCKETS_TIMEOUT)
	defer cancel()

	c, _, err := websocket.Dial(ctx, knownNode.URL, nil)
	if err != nil {
		return nil, err
	}

	if wsClient.conn, err = websockets.NewConnection(c, knownNode.URL, knownNode, false); err != nil {
		return nil, err
	}

	return wsClient, nil
}
