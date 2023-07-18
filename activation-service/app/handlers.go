package app

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"gopkg.in/validator.v2"
)

var errInternalServer = errors.New("internal server error")

// ActivationInput struct for data needed while activation
type ActivationInput struct {
	KYCSignature       string         `json:"kycSignature"`
	Data               ActivationData `json:"data"`
	SubstrateAccountID string         `json:"substrateAccountID" binding:"required" validate:"nonzero"`
}

// ActivationData struct for data needed while activation
type ActivationData struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
}

func (a *App) activateHandler(r *http.Request, w http.ResponseWriter) (interface{}, Response) {
	var input ActivationInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, badRequest(errors.New("failed to read input data"))
	}

	err = validator.Validate(input)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, badRequest(errors.New("invalid input data"))
	}

	account, err := substrate.FromAddress(string(input.SubstrateAccountID))
	if err != nil {
		log.Error().Err(err).Send()
		return nil, notFound(errors.New("substrate account is not found"))
	}

	balance, err := a.substrateConn.GetBalance(account)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, internalServerError(errInternalServer)
	}

	if balance.Free.Uint64() == 0 {
		err = a.substrateConn.Transfer(a.identity, a.config.ActivationAmount*1e7, account)
		if err != nil {
			log.Error().Err(err).Send()
			return nil, internalServerError(errInternalServer)
		}
	}

	if balance.Free.Uint64() < 15000 {
		err = a.substrateConn.Transfer(a.identity, 15000, account)
		if err != nil {
			log.Error().Err(err).Send()
			return nil, internalServerError(errInternalServer)
		}
	}

	return nil, ok()
}
