package peer

import (
	"crypto/md5"
	"fmt"
	"io"

	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer/types"
)

func Challenge(env *types.Envelope) ([]byte, error) {
	hash := md5.New()
	if err := challenge(hash, env); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

func challenge(w io.Writer, env *types.Envelope) error {
	if _, err := fmt.Fprintf(w, "%s", env.Uid); err != nil {
		return err
	}
	if env.Tags != nil {
		if _, err := fmt.Fprintf(w, "%s", *env.Tags); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "%d", env.Timestamp); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", env.Expiration); err != nil {
		return err
	}

	if err := challengeAddress(w, env.Source); err != nil {
		return err
	}

	if err := challengeAddress(w, env.Destination); err != nil {
		return err
	}

	var err error
	if request := env.GetRequest(); request != nil {
		err = challengeRequest(w, request)
	} else if response := env.GetResponse(); response != nil {
		err = challengeResponse(w, response)
	} else if envErr := env.GetError(); envErr != nil {
		err = challengeError(w, envErr)
	}

	if err != nil {
		return err
	}

	if env.Schema != nil {
		if _, err := fmt.Fprintf(w, "%s", *env.Schema); err != nil {
			return err
		}
	}

	if env.Federation != nil {
		if _, err := fmt.Fprintf(w, "%s", *env.Federation); err != nil {
			return err
		}
	}

	// data is always hashed as is either if it's
	// a plain data (not encrypted)
	// or a cipher
	var data []byte
	if plain := env.GetPlain(); plain != nil {
		data = plain
	} else if cipher := env.GetCipher(); cipher != nil {
		data = cipher
	}

	if data != nil {
		if _, err := w.Write(data); err != nil {
			return err
		}
	}

	return nil
}

func challengeAddress(w io.Writer, addr *types.Address) error {
	if addr == nil {
		return nil
	}
	if _, err := fmt.Fprintf(w, "%d", addr.Twin); err != nil {
		return err
	}

	if addr.Connection != nil {
		if _, err := fmt.Fprintf(w, "%s", *addr.Connection); err != nil {
			return err
		}
	}

	return nil
}

func challengeRequest(w io.Writer, request *types.Request) error {
	if request == nil {
		return nil
	}

	if _, err := fmt.Fprintf(w, "%s", request.Command); err != nil {
		return err
	}
	return nil
}

func challengeResponse(w io.Writer, response *types.Response) error {
	return nil

}

func challengeError(w io.Writer, err *types.Error) error {
	if err == nil {
		return nil
	}
	if _, err := fmt.Fprintf(w, "%d", err.Code); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", err.Message); err != nil {
		return err
	}

	return nil
}
