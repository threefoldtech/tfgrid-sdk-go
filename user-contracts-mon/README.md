# TFGrid Contract Monitor Bot

## Overview

The Contract Monitor Bot is a tool designed to monitor user contracts within ThreeFold Grid.

**Features**:

- Monitors user contracts and nodes status.
- Customizable alerting and notification system.

## Getting Started

### Prerequisites

Ensure that you have installed:

- Go programming language (version 1.19 or higher)
- Git

### How to start

1. Clone this repository to your local machine:

   ```bash
   git clone https://github.com/threefoldtech/tfgrid-sdk-go.git
   cd tfgrid-sdk-go/user-contracts-mon
   ```

2. Setup your telegram bot and your env

   - Create a new [telegram bot](README.md#create-a-bot) if you don't have.
   - Create a new env file `.env`, for example:

     ```env
        BOT_TOKEN=<your bot token>
        MNEMONIC=<your mnemonics>
        NETWORK=<main, dev, test, qa>
        INTERVAL=<number of hours between notifications>
        ```

3. Run the bot:

   - Using go

     ```bash
        go run main.go -e .env
        ```

   - Using Docker

     ```bash
        docker build -t contract-mon .
        docker run -it contract-mon -e env=.env
        ```

## Create a bot

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
