//go:build !wasm
// +build !wasm

package arguments

//use spaces for default https://github.com/docopt/docopt.go/issues/57

const commands = `PANDORA CASH.

Usage:
  pandorapay [--pprof] [--network=network] [--debug] [--gui-type=type] [--forging] [--new-devnet] [--run-testnet-script] [--node-name=name] [--tcp-server-port=port] [--tcp-server-address=address] [--tcp-server-auto-tls-certificate] [--tcp-server-tls-cert-file=path] [--tcp-server-tls-key-file=path] [--instance=prefix] [--instance-id=id] [--set-genesis=genesis] [--create-new-genesis=args] [--store-wallet-type=type] [--store-chain-type=type] [--node-consensus=type] [--tcp-max-clients=limit] [--tcp-max-server-sockets=limit] [--node-provide-extended-info-app=bool] [--wallet-encrypt=args] [--wallet-decrypt=password] [--wallet-remove-encryption] [--wallet-export-shared-staked-address=args] [--wallet-import-secret-mnemonic=mnemonic] [--wallet-import-secret-entropy=entropy] [--hcaptcha-secret=args] [--faucet-testnet-enabled=args] [--delegator-enabled=bool] [--delegator-require-auth=bool] [--delegates-maximum=args] [--auth-users=args] [--light-computations] [--balance-decrypter-disable-init] [--balance-decrypter-table-size=size] [--tcp-connections-ready=threshold] [--exit] [--skip-init-sync] [--tcp-server-url=url] [--tcp-proxy=PROXY]
  pandorapay -h | --help
  pandorapay -v | --version

Options:
  -h --help                                          Show this screen.
  --version                                          Show version.
  --gui-type=type                                    GUI format. Accepted values: "interactive|non-interactive".  [default: interactive]
  --debug                                            Debug mode enabled (print log message).
  --instance=prefix                                  Prefix of the instance [default: 0].
  --instance-id=id                                   Number of forked instance (when you open multiple instances). It should be a string number like "1","2","3","4" etc
  --network=network                                  Select network. Accepted values: "mainnet|testnet|devnet". [default: mainnet]
  --new-devnet                                       Create a new devnet genesis.
  --run-testnet-script                               Run testnet script which will create dummy transactions in the network.
  --set-genesis=genesis                              Manually set the Genesis via a JSON. By using argument "file" it will read it via a file.
  --create-new-genesis=args                          Create a new Genesis. Useful for creating a new private testnet. Argument must be "0.stake,1.stake,2.stake"
  --store-wallet-type=type                           Set Wallet Store Type. Accepted values: "bolt|bunt|bunt-memory|memory". [default: bolt]
  --store-chain-type=type                            Set Chain Store Type. Accepted values: "bolt|bunt|bunt-memory|memory".  [default: bolt]
  --forging                                          Start Forging blocks.
  --node-name=name                                   Change node name.
  --node-consensus=type                              Consensus type. Accepted values: "full|app|none" [default: full].
  --node-provide-extended-info-app=bool              Storing and serving additional info to wallet nodes. [default: true]. To enable, it requires full node
  --tcp-server-url=url                               TCP Server URL (schema, address, port, path).
  --tcp-server-port=port                             Change node tcp server port [default: 8080].
  --tcp-max-clients=limit                            Change limit of clients [default: 50].
  --tcp-max-server-sockets=limit                     Change limit of servers [default: 500].
  --tcp-connections-ready=threshold                  Number of connections to become "ready" state [default: 1].
  --tcp-server-address=address                       Change node tcp address.
  --tcp-server-auto-tls-certificate                  If no certificate.crt is provided, this option will generate a valid TLS certificate via autocert package. You still need a valid domain provided and set --tcp-server-address.
  --tcp-server-tls-cert-file=path                    Load TLS certificate file from given path.
  --tcp-server-tls-key-file=path                     Load TLS ke file from given path.
  --tcp-proxy=proxy                                  Proxy used for network.
  --wallet-import-secret-mnemonic=mnemonic           Import Wallet from a given Mnemonic. It will delete your existing wallet. 
  --wallet-import-secret-entropy=entropy             Import Wallet from a given Entropy. It will delete your existing wallet.
  --wallet-encrypt=args                              Encrypt wallet. Argument must be "password,difficulty".
  --wallet-decrypt=password                          Decrypt wallet.
  --wallet-remove-encryption                         Remove wallet encryption.
  --wallet-export-shared-staked-address=args         Derive and export Staked address. Argument must be "account,nonce,path".
  --hcaptcha-secret=args                             hcaptcha Secret.
  --faucet-testnet-enabled=args                      Enable Faucet Testnet. Use "true" to enable it
  --delegator-enabled=bool                           Enable Delegator. Will allow other users to Delegate to the node. Use "true" to enable it
  --delegator-require-auth=bool                      Delegator will require authentication.
  --delegates-maximum=args                           Maximum number of Delegates
  --auth-users=args                                  Credential for Authenticated Users. Arguments must be a JSON "[{'user': 'username', 'pass': 'secret'}]".
  --light-computations                               Reduces the computations for a testnet node.
  --balance-decrypter-disable-init                   Disable first balance decrypter initialization. 
  --balance-decrypter-table-size=size                Balance Decrypter initial table size. [default: 23]
  --exit                                             Exit node.
  --skip-init-sync                                   Skip sync wait at when the node started. Useful when creating a new testnet.
`
