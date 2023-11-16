package config

import (
	"net/url"
	"pandora-pay/config/arguments"
)

var (
	TCP_PROXY                  = ""
	TCP_PROXY_URL              *url.URL
	TCP_PROXY_BYPASS_LOCALHOST = false
)

func initNetworkConfig() (err error) {

	if arguments.Arguments["--tcp-proxy"] != nil {
		TCP_PROXY = arguments.Arguments["--tcp-proxy"].(string)
		if TCP_PROXY_URL, err = url.Parse(TCP_PROXY); err != nil {
			return err
		}
	}

	if arguments.Arguments["--tcp-proxy-bypass-localhost"] != nil {
		TCP_PROXY_BYPASS_LOCALHOST = true
	}

	return nil
}
