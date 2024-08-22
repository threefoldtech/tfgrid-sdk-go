package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/melbahja/goph"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-compose/internal/types"
	"golang.org/x/crypto/ssh"
)

// verifyHost verifies the host key of the server.
func verifyHost(host string, remote net.Addr, key ssh.PublicKey) error {
	// Check if the host is in known hosts file.
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

// runHealthCheck runs the health check on the VM
func runHealthCheck(healthCheck types.HealthCheck, user, ipAddr string) error {
	var auth goph.Auth
	var err error

	if goph.HasAgent() {
		log.Info().Msg("using ssh agent for authentication")
		auth, err = goph.UseAgent()
	} else {
		log.Info().Msg("using private key for authentication")
		var sshDir, privKeyPath string
		sshDir, err = getUserSSHDir()
		if err != nil {
			return err
		}
		privKeyPath = filepath.Join(sshDir, "id_rsa")
		auth, err = goph.Key(privKeyPath, "")
	}

	if err != nil {
		return err
	}

	timeoutDuration, err := time.ParseDuration(healthCheck.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout format %w", err)
	}

	startTime := time.Now()
	var client *goph.Client

	for {
		elapsedTime := time.Since(startTime)
		if elapsedTime >= timeoutDuration {
			return fmt.Errorf("timeout reached while waiting for SSH connection")
		}

		remainingTime := timeoutDuration - elapsedTime

		client, err = goph.NewConn(&goph.Config{
			User:     user,
			Port:     22,
			Addr:     ipAddr,
			Auth:     auth,
			Callback: verifyHost,
		})

		if err == nil {
			defer client.Close()
			break
		}

		log.Info().Err(err).Msg("ssh connection attempt failed, retrying...")
		time.Sleep(time.Second)

		if remainingTime < time.Second {
			time.Sleep(remainingTime)
		}
	}

	command := strings.Join(healthCheck.Test, " ")

	intervalDuration, err := time.ParseDuration(healthCheck.Interval)
	if err != nil {
		return fmt.Errorf("invalid interval format %w", err)
	}

	for i := 0; i < int(healthCheck.Retries); i++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		out, err := runCommandWithContext(ctx, client, command)
		if err == nil {
			log.Info().Msgf("health check succeeded %s", string(out))
			return nil
		}

		log.Info().Str("attempt", fmt.Sprintf("%d/%d", i+1, healthCheck.Retries)).Err(err).Msg("health check failed, retrying...")
		time.Sleep(intervalDuration)
	}

	return fmt.Errorf("health check failed after %d retries %w", healthCheck.Retries, err)
}

// runCommandWithContext runs the command on the client with context and returns the output and error
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

// getUserSSHDir returns the path to the user's SSH directory(e.g. ~/.ssh)
func getUserSSHDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".ssh"), nil
}
