#!/bin/sh

git clone $REPO_URL repo && cd repo
ginit start -f ./Procfile -e ./.env

