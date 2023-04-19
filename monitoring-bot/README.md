# tfgrid monitoring bot

<a href='https://github.com/jpoles1/gopherbadger' target='_blank'>![gopherbadger-tag-do-not-edit](https://img.shields.io/badge/Go%20Coverage-48%25-brightgreen.svg?longCache=true&style=flat)</a>
[![Testing](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/monbot-test.yml/badge.svg?branch=development_mono)](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/monbot-test.yml) [![Testing](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/monbot-lint.yml/badge.svg?branch=development_mono)](https://github.com/threefoldtech/tfgrid-sdk-go/actions/workflows/monbot-lint.yml)

This is a bot to monitor the balance in accounts and send warnings if it is under some threshold.
It also monitors the behavior of the rmb proxy

## How to start

-   Create a new [telegram bot](README.md#create-a-bot-if-you-dont-have) if you don't have.
-   Create a new env file `.env`, for example:

```env
TESTNET_MNEMONIC=<your mainnet mnemonic>
MAINNET_MNEMONIC=<your testnet mnemonic>
DEVNET_MNEMONIC=<your devnet mnemonic>
QANET_MNEMONIC=<your qanet mnemonic>
DEV_FARM_NAME=Freefarm
QA_FARM_NAME=Freefarm
MAIN_FARM_NAME=Freefarm
TEST_FARM_NAME=FreeFarm
BOT_TOKEN=<your token>
CHAT_ID=<your chat ID>
MINS=<number of minutes between each message>
```

-   Create a new json file `wallets.json` and add the list of addresses you want to monitor, for example:

```json
{ 
    "testnet": [{ 
        "name": "<your wallet name>", 
        "address": "<your tfchain address>", 
        "threshold": 700 
    }],

    "mainnet": [{ 
        "name": "<your wallet name>", 
        "address": "<your tfchain address>", 
        "threshold": 700 
    }]
}
```

-   Get the binary

> Download the latest from the [releases page](https://github.com/threefoldtech/tfgrid-sdk-go/monitoring-bot/releases)

-   Run the bot

After downloading the binary

```bash
sudo cp monitoring-bot /usr/local/bin
monitoring-bot -e .env -w wallets.json
```

Where

-   `.env` is the environment file
-   `wallets.json` is the json file of wallets to be monitored  

## Create a bot if you don't have

-   Open telegram app
-   Create a new bot
  
```ordered
1. Find telegram bot named "@botfarther"
2. Type /newbot
```

-   Get the bot token
  
```ordered
1. In the same bot named "@botfarther"
2. Type /token
3. Choose your bot
```

-   Get your chat ID

```ordered
1. Search for @RawDataBot and select Telegram Bot Raw from the drop-down list.
2. In the json returned, you will find it in section message -> chat -> id
```

## Test

```bash
make test
```
