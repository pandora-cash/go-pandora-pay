//go:build !js
// +build !js

package websock

import (
	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"pandora-pay/config"
)

type Conn struct {
	*websocket.Conn
}

func Dial(URL string) (*Conn, error) {

	//tcp proxy
	useProxy := false

	var dialer *websocket.Dialer
	if config.TCP_PROXY_URL != nil {

		useProxy = true

		if config.TCP_PROXY_BYPASS_LOCALHOST {
			u, err := url.Parse(URL)
			if err != nil {
				return nil, err
			}
			hostname := u.Hostname()
			if hostname == "localhost" || hostname == "127.0.0.1" {
				useProxy = false
			}
		}

	}

	if useProxy {
		netDialer, err := proxy.FromURL(config.TCP_PROXY_URL, &net.Dialer{})
		if err != nil {
			return nil, err
		}
		dialer = &websocket.Dialer{NetDial: netDialer.Dial}
	} else {
		dialer = websocket.DefaultDialer
	}

	c, _, err := dialer.Dial(URL, nil)
	if err != nil {
		return nil, err
	}

	return &Conn{c}, nil
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
} // use default options

func Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	return &Conn{c}, nil
}
