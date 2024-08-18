package deploy

import (
	"context"
	"fmt"

	"net"
	"strings"
	"time"

	"github.com/melbahja/goph"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
	"golang.org/x/crypto/ssh"
)

func verifyHost(host string, remote net.Addr, key ssh.PublicKey) error {

	//
	// If you want to connect to new hosts.
	// here your should check new connections public keys
	// if the key not trusted you shuld return an error
	//

	// hostFound: is host in known hosts file.
	// err: error if key not in known hosts file OR host in known hosts file but key changed!
	hostFound, err := goph.CheckKnownHost(host, remote, key, "")

	// Host in known hosts but key mismatch!
	// Maybe because of MAN IN THE MIDDLE ATTACK!
	if hostFound && err != nil {

		return err
	}

	// handshake because public key already exists.
	if hostFound && err == nil {
		return nil
	}

	// Add the new host to known hosts file.
	return goph.AddKnownHost(host, remote, key, "")
}

// needs refactoring
func runHealthCheck(healthCheck types.HealthCheck, privKeyPath, user, ipAddr string) error {
	auth, err := goph.Key(privKeyPath, "")
	if err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	var client *goph.Client
	for i := 0; i < 5; i++ {
		client, err = goph.NewConn(&goph.Config{
			User:     user,
			Port:     22,
			Addr:     ipAddr,
			Auth:     auth,
			Callback: verifyHost,
		})

		if err == nil {
			break
		}

		log.Info().Str("attempt", fmt.Sprintf("%d", i+1)).Err(err).Msg("ssh connection attempt failed, retrying...")
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to establish ssh connection after retries %w", err)
	}

	defer client.Close()

	command := strings.Join(healthCheck.Test, " ")

	intervalDuration, err := time.ParseDuration(healthCheck.Interval)
	if err != nil {
		return fmt.Errorf("invalid interval format %w", err)
	}

	timeoutDuration, err := time.ParseDuration(healthCheck.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout format %w", err)
	}

	var out []byte
	for i := 0; i < int(healthCheck.Retries); i++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		out, err = runCommandWithContext(ctx, client, command)
		if err == nil {
			log.Info().Msgf("health check succeeded %s", string(out))
			return nil
		}

		log.Info().Str("attempt", fmt.Sprintf("%d/%d", i+1, healthCheck.Retries)).Err(err).Msg("health check failed, retrying...")
		time.Sleep(intervalDuration)
	}

	return fmt.Errorf("health check failed after %d retries %w", healthCheck.Retries, err)
}

func runCommandWithContext(ctx context.Context, client *goph.Client, command string) ([]byte, error) {
	done := make(chan struct{})
	var out []byte
	var err error

	go func() {
		out, err = client.Run(command)
		close(done)
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("command timed out")
	case <-done:
		return out, err
	}
}
