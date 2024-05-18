# Developer guidelines

## Developer setup

- make sure to have `git`, `make`, `docker` installed

## Before committing code

- Make sure to enable codacy for the base branch your are using.
- Make sure to run the preliminary checks `fmt` `lint` `cyclo` `deadcode` `spelling` `staticcheck`, using the command `make checks`.
- Make sure tests pass using `make test`, for grid-proxy run `make test-unit`.
