// Package integration for integration tests
package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	mrand "math/rand"
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"golang.org/x/crypto/ssh"
)

const (
	ubuntuFlist = "https://hub.grid.tf/tf-official-apps/threefoldtech-ubuntu-22.04.flist"
)

func setup() (deployer.TFPluginClient, error) {
	mnemonics := os.Getenv("MNEMONICS")
	log.Printf("mnemonics: %s", mnemonics)

	network := os.Getenv("NETWORK")
	log.Printf("network: %s", network)

	return deployer.NewTFPluginClient(mnemonics, deployer.WithNetwork(network), deployer.WithLogs())
}

func generateBasicNetwork(nodeIDs []uint32) (workloads.ZNet, error) {
	myCeliumKeys := make(map[uint32][]byte)
	for _, nodeID := range nodeIDs {
		key, err := workloads.RandomMyceliumKey()
		if err != nil {
			return workloads.ZNet{}, fmt.Errorf("could not create mycelium key: %v", err)
		}
		myCeliumKeys[nodeID] = key
	}
	return workloads.ZNet{
		Name:        fmt.Sprintf("net_%s", generateRandString(10)),
		Description: "network for testing",
		Nodes:       nodeIDs,
		IPRange: zos.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		MyceliumKeys: myCeliumKeys,
	}, nil
}

// CheckConnection used to test connection
func CheckConnection(addr string, port string) bool {
	for t := time.Now(); time.Since(t) < 3*time.Second; {
		con, err := net.DialTimeout("tcp", net.JoinHostPort(addr, port), time.Second*12)
		if err == nil {
			con.Close()
			return true
		}
	}
	return false
}

// RemoteRun used for running cmd remotely using ssh
func RemoteRun(user string, addr string, cmd string, privateKey string) (string, error) {
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return "", errors.Wrapf(err, "could not parse ssh private key %v", key)
	}
	// Authentication
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
	}

	// Connect
	port := "22"
	client, err := ssh.Dial("tcp", net.JoinHostPort(addr, port), config)
	if err != nil {
		return "", errors.Wrapf(err, "could not start ssh connection")
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", errors.Wrapf(err, "could not create new session with message error")
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", errors.Wrapf(err, "could not execute command on remote with output %s", output)
	}
	return string(output), nil
}

// GenerateSSHKeyPair creates the public and private key for the machine
func GenerateSSHKeyPair() (string, string, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return "", "", errors.Wrapf(err, "could not generate rsa key")
	}

	pemKey := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaKey)}
	privateKey := pem.EncodeToMemory(pemKey)

	pub, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return "", "", errors.Wrapf(err, "could not extract public key")
	}
	authorizedKey := ssh.MarshalAuthorizedKey(pub)
	return string(authorizedKey), string(privateKey), nil
}

func generateRandString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[mrand.Intn(len(letters))]
	}
	return string(b)
}

func generateBasicVM(vmName string, nodeID uint32, networkName string, publicKey string) (workloads.VM, error) {
	seed, err := workloads.RandomMyceliumIPSeed()
	if err != nil {
		return workloads.VM{}, err
	}
	return workloads.VM{
		Name:        vmName,
		NodeID:      nodeID,
		NetworkName: networkName,
		CPU:         minCPU,
		MemoryMB:    minMemory * 1024,
		Flist:       "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		Entrypoint:  "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
		MyceliumIPSeed: seed,
	}, nil
}
