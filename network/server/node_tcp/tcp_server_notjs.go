//go:build !wasm
// +build !wasm

package node_tcp

import (
	"crypto/tls"
	"errors"
	"golang.org/x/crypto/acme/autocert"
	"net"
	"net/http"
	"net/url"
	"os"
	"pandora-pay/config"
	"pandora-pay/config/arguments"
	"pandora-pay/gui"
	"pandora-pay/helpers/recovery"
	"pandora-pay/network/banned_nodes"
	"pandora-pay/network/network_config"
	"pandora-pay/network/server/node_http"
	"path"
	"strconv"
	"time"
)

type tcpServerType struct {
	Address     string
	Port        string
	URL         *url.URL
	tcpListener net.Listener
}

var TcpServer *tcpServerType

func NewTcpServer() error {

	TcpServer = &tcpServerType{}

	// Create local listener on next available port

	port := arguments.Arguments["--tcp-server-port"].(string)

	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return errors.New("Port is not a valid port number")
	}

	portNumber += config.INSTANCE_ID

	port = strconv.Itoa(portNumber)

	var address string
	if arguments.Arguments["--tcp-server-address"] != nil {
		address = arguments.Arguments["--tcp-server-address"].(string)
	}

	TcpServer.Port = port

	shareAddress := true
	if address == "na" {
		shareAddress = false
		address = ""
	}

	if shareAddress {
		if address == "" {
			conn, err := net.Dial("udp", "8.8.8.8:80")
			if err != nil {
				return errors.New("Error dialing dns to discover my own ip" + err.Error())
			}
			address = conn.LocalAddr().(*net.UDPAddr).IP.String()
			if err = conn.Close(); err != nil {
				return errors.New("Error closing the connection" + err.Error())
			}
		}
	}

	banned_nodes.BannedNodes.BanURL(&url.URL{Scheme: "ws", Host: address + ":" + port, Path: "/ws"}, "You can't connect to yourself", 10*365*24*time.Hour)

	var certPath, keyPath string
	if arguments.Arguments["--tcp-server-tls-cert-file"] != nil {
		certPath = arguments.Arguments["--tcp-server-tls-cert-file"].(string)
	} else {
		certPath = path.Join(config.ORIGINAL_PATH, "certificate.crt")
	}

	if arguments.Arguments["--tcp-server-tls-key-file"] != nil {
		keyPath = arguments.Arguments["--tcp-server-tls-key-file"].(string)
	} else {
		keyPath = path.Join(config.ORIGINAL_PATH, "certificate.key")
	}

	var tlsConfig *tls.Config
	if _, err = os.Stat(certPath); err == nil {
		cer, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return err
		}
		tlsConfig = &tls.Config{Certificates: []tls.Certificate{cer}}
	} else if arguments.Arguments["--tcp-server-auto-tls-certificate"] == true {

		if arguments.Arguments["--tcp-server-address"] == "" {
			return errors.New("To get an automatic Automatic you need to specify a domain --tcp-server-address=\"domain.com\"")
		}

		cache := path.Join(config.ORIGINAL_PATH, "certManager")

		if _, err = os.Stat(cache); os.IsNotExist(err) {
			if err = os.Mkdir(cache, 0755); err != nil {
				return err
			}
		}

		// create the autocert.Manager with domains and path to the cache
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(address),
			Cache:      autocert.DirCache(cache), //it is designed to avoid generating multiple certificates for the same instance
		}

		tlsConfig = certManager.TLSConfig()

	}

	if shareAddress {

		var u *url.URL

		if arguments.Arguments["--tcp-server-url"] != nil {
			if u, err = url.Parse(arguments.Arguments["--tcp-server-url"].(string)); err != nil {
				return err
			}
		} else {
			u = &url.URL{Scheme: "http", Host: address + ":" + port, Path: ""}
			if tlsConfig != nil {
				u.Scheme += "s"
			}
		}

		websocketUrl := &url.URL{Scheme: u.Scheme, Host: u.Host, Path: u.Path}
		if websocketUrl.Scheme == "http" {
			websocketUrl.Scheme = "ws"
		} else if websocketUrl.Scheme == "https" {
			websocketUrl.Scheme = "wss"
		}
		websocketUrl.Path += "/ws"

		network_config.NETWORK_ADDRESS_URL_STRING = u.String()
		network_config.NETWORK_WEBSOCKET_ADDRESS_URL_STRING = websocketUrl.String()

		banned_nodes.BannedNodes.BanURL(websocketUrl, "You can't connect to yourself", 10*365*24*time.Hour)
		TcpServer.URL = u
		TcpServer.Address = u.Host
	}

	if tlsConfig != nil {
		if TcpServer.tcpListener, err = tls.Listen("tcp", ":"+port, tlsConfig); err != nil {
			return err
		}
		gui.GUI.Info("TLS Certificate loaded for ", address, port)
	} else {
		// no ssl at all
		if TcpServer.tcpListener, err = net.Listen("tcp", ":"+port); err != nil {
			return errors.New("Error creating TcpServer" + err.Error())
		}
		gui.GUI.Warning("No TLS Certificate")
	}

	gui.GUI.InfoUpdate("TCP", address+":"+port)

	if err = node_http.NewHttpServer(); err != nil {
		return err
	}

	recovery.SafeGo(func() {
		if err := http.Serve(TcpServer.tcpListener, *node_http.HttpServer.GetHttpHandler()); err != nil {
			gui.GUI.Error("Error opening HTTP server", err)
		}
		gui.GUI.Info("HTTP server")
	})

	return nil
}
