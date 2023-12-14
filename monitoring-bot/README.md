# tfgrid monitoring bot

<a href='https://github.com/jpoles1/gopherbadger' target='_blank'>![gopherbadger-tag-do-not-edit](https://img.shields.io/badge/Go%20Coverage-39%25-brightgreen.svg?longCache=true&style=flat)</a>

This is a bot to monitor some TFGrid functionalities here is a list:

- liveness of gridproxy on different networks and rmb call behavior to some farmer nodes. If all selected farmer nodes failed for a specific network the bot will send an alert message otherwise It won't send any messages.
- the balance in accounts and send warnings if it is under some threshold.
- transactions to/from stellar bridge.

## How to start

- Create a new [telegram bot](README.md#create-a-bot-if-you-dont-have) if you don't have.
- Create a new env file `.env`, for example:

    ```env
    TESTNET_MNEMONIC=<your mainnet mnemonic>
    MAINNET_MNEMONIC=<your testnet mnemonic>
    DEVNET_MNEMONIC=<your devnet mnemonic>
    QANET_MNEMONIC=<your qanet mnemonic>
    DEV_FARM_NAME=Freefarm
    QA_FARM_NAME=Freefarm
    MAIN_FARM_NAME=Freefarm
    TEST_FARM_NAME=FreeFarm
    BOT_TOKEN=<bot token. you got it after creating the bot>
    CHAT_ID=<your personal chat ID, where bot will send you >
    MINS=<number of minutes between each message>
    PUBLIC_STELLAR_SECRET=<stellar account secret on stellar public network>
    PUBLIC_STELLAR_ADDRESS=<stellar account address on stellar public network>
    TEST_STELLAR_SECRET=<stellar account secret on stellar test network>
    TEST_STELLAR_ADDRESS=<stellar account address on stellar test network>
    ```
| Note: wallets on Stellar should have some lumens for the fees, stellar charges about 0.00001 xlm per txn.

- Create a new json file `wallets.json` and add the list of addresses you want to monitor, for example:

    ```json
    {
    "testnet": [
        {
        "name": "<your wallet name>",
        "address": "<your tfchain address>",
        "threshold": 700
        }
    ],

    "mainnet": [
        {
        "name": "<your wallet name>",
        "address": "<your tfchain address>",
        "threshold": 700
        }
    ]
    }
    ```

- Run the bot:

  - From the src:
  
    ```bash
    go run main.go -e .env -w wallets.json
    ```

  - From the release binary:
    > Download the latest from the [releases page](https://github.com/threefoldtech/tfgrid-sdk-go/releases)

    ```bash
    sudo cp monitoring-bot /usr/local/bin
    monitoring-bot -e .env -w wallets.json
    ```

    Where

    - `.env` is the environment file
    - `wallets.json` is the json file of wallets to be monitored

## Create a bot if you don't have

- Open telegram app
- Create a new bot

    ```ordered
    1. Find telegram bot named "@botfarther"
    2. Type /newbot
    ```

- Get the bot token

    ```ordered
    1. In the same bot named "@botfarther"
    2. Type /token
    3. Choose your bot
    ```

- Get your chat ID

    ```ordered
    1. Search for @RawDataBot and select Telegram Bot Raw from the drop-down list.
    2. In the json returned, you will find it in section message -> chat -> id
    ```

### Build

```bash
make build
```

## Test

```bash
make test
```
