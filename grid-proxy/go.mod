module github.com/threefoldtech/tfgrid-sdk-go/grid-proxy

go 1.21

require (
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/go-acme/lego/v4 v4.16.1
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/schema v1.4.1
	github.com/lib/pq v1.10.9
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.33.0
	github.com/stretchr/testify v1.9.0
	github.com/swaggo/http-swagger v1.3.4
	github.com/swaggo/swag v1.16.3
	github.com/threefoldtech/tfchain/clients/tfchain-client-go v0.0.0-20240827163226-d4e15e206974
	github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go v0.15.14
	github.com/threefoldtech/zos v0.5.6-0.20240902110349-172a0a29a6ee
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa
	gorm.io/driver/postgres v1.5.7
	gorm.io/gorm v1.25.12

)

require (
	github.com/ChainSafe/go-schnorrkel v1.1.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/centrifuge/go-substrate-rpc-client/v4 v4.0.12 // indirect
	github.com/cosmos/go-bip39 v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/decred/base58 v1.0.5 // indirect
	github.com/decred/dcrd/crypto/blake256 v1.0.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/ethereum/go-ethereum v1.11.6 // indirect
	github.com/go-jose/go-jose/v4 v4.0.1 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/spec v0.20.8 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/gtank/merlin v0.1.1 // indirect
	github.com/gtank/ristretto255 v0.1.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/holiman/uint256 v1.2.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.4.3 // indirect
	github.com/jbenet/go-base58 v0.0.0-20150317085156-6237cf65f3a6 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/miekg/dns v1.1.58 // indirect
	github.com/mimoo/StrobeGo v0.0.0-20220103164710-9a04d6ca976b // indirect
	github.com/pierrec/xxHash v0.1.5 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rs/cors v1.10.1 // indirect
	github.com/swaggo/files v1.0.1 // indirect
	github.com/threefoldtech/zbus v1.0.1 // indirect
	github.com/vedhavyas/go-subkey v1.0.3 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.23.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	gonum.org/v1/gonum v0.15.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go => ../rmb-sdk-go
