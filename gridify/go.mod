module github.com/threefoldtech/tfgrid-sdk-go/gridify

go 1.21

require (
	github.com/cosmos/go-bip39 v1.0.0
	github.com/go-git/go-git/v5 v5.12.0
	github.com/golang/mock v1.6.0
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.33.0
	github.com/spf13/cobra v1.8.1
	github.com/stretchr/testify v1.9.0
	github.com/threefoldtech/tfgrid-sdk-go/grid-client v0.15.10
	github.com/threefoldtech/tfgrid-sdk-go/grid-proxy v0.15.10
)

require (
	dario.cat/mergo v1.0.0 // indirect
	github.com/ChainSafe/go-schnorrkel v1.1.0 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/ProtonMail/go-crypto v1.0.0 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/centrifuge/go-substrate-rpc-client/v4 v4.0.12 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/decred/base58 v1.0.5 // indirect
	github.com/decred/dcrd/crypto/blake256 v1.0.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/ethereum/go-ethereum v1.11.6 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.5.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/schema v1.3.0 // indirect
	github.com/gorilla/websocket v1.5.2 // indirect
	github.com/gtank/merlin v0.1.1 // indirect
	github.com/gtank/ristretto255 v0.1.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/holiman/uint256 v1.2.3 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-base58 v0.0.0-20150317085156-6237cf65f3a6 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mimoo/StrobeGo v0.0.0-20220103164710-9a04d6ca976b // indirect
	github.com/pierrec/xxHash v0.1.5 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rs/cors v1.10.1 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/skeema/knownhosts v1.2.2 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/threefoldtech/tfchain/clients/tfchain-client-go v0.0.0-20240416134707-eb36aa90df2d // indirect
	github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go v0.15.10 // indirect
	github.com/threefoldtech/zos v0.5.6-0.20240613101720-0a4726af4edd // indirect
	github.com/vedhavyas/go-subkey v1.0.3 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20200609130330-bd2cb7843e1b // indirect
	gonum.org/v1/gonum v0.15.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/threefoldtech/tfgrid-sdk-go/grid-client => ../grid-client

replace github.com/threefoldtech/tfgrid-sdk-go/grid-proxy => ../grid-proxy

replace github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go => ../rmb-sdk-go
