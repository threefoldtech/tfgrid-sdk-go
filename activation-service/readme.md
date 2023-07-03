# Substrate funding service

A TFChain Wallet account requires a minimum balance to exist and function. New TFChain users will not automatically have any tokens (also not on stellar).
 Therefore an activation service for new TFChain wallets is created. It activates new TFChain wallet addresses by depositing a minimal amount of TFT (currently 1 TFT).

## Installing and running

create `.env` file with following content:

```bash
URL=wss://substrate01.threefold.io
MNEMONIC=substrate ed25519 private words
ACTIVATION_AMOUNT=1
```

Run backend

```bash
make run
```

## Endpoints

### Activate

`/activation/activate`

Activates a Substrate account and puts 500 tokens on it.

Example: Post to `localhost:3000/activation/activate`

```sh
curl --header "Content-Type: application/json" \
  --request POST \
  --data '{"substrateAccountID": "some_id"}' \
  http://localhost:3000/activation/activate
```

## Networks

We will run an activation service for each TF Grid network (mainnet, testnet, devnet).
