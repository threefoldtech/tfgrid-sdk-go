package parser

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/cosmos/go-bip39"
	deployer "github.com/threefoldtech/tfgrid-sdk-go/mass-deployer/pkg/mass-deployer"
)

func validateMnemonic(mnemonic string) error {
	if !bip39.IsMnemonicValid(mnemonic) {
		return fmt.Errorf("invalid mnemonic: %s", mnemonic)
	}
	return nil
}

func validateNetwork(network string) error {
	networks := []string{"dev", "test", "qa", "main"}
	if !slices.Contains(networks, network) {
		return fmt.Errorf("invalid network: %s, network can be one of %+v", network, networks)
	}
	return nil
}

func validateVMs(vms []deployer.Vms, nodeGroups []string, sskKeys map[string]string) error {
	for _, vm := range vms {
		if !slices.Contains(nodeGroups, strings.TrimSpace(vm.Nodegroup)) {
			return fmt.Errorf("invalid node_group: %s in vms group: %s", vm.Nodegroup, vm.Name)
		}
		if _, ok := sskKeys[vm.SSHKey]; !ok {
			return fmt.Errorf("vms group %s ssh_key is invalid, should be valid name refers to one of ssh_keys map", vm.Name)
		}
		if err := validateFlist(vm.Flist, vm.Name); err != nil {
			return err
		}

	}
	return nil
}

func validateFlist(flist, name string) error {
	flistExt := path.Ext(flist)
	if flistExt != ".fl" && flistExt != ".flist" {
		return fmt.Errorf("vms group %s flist: %s is invalid, should be valid flist", name, flist)
	}

	if flistExt == ".flist" {
		hash := md5.Sum([]byte(flist + ".md5"))
		response, err := http.Get(flist + fmt.Sprintf("%x", hash))
		if err != nil {
			return fmt.Errorf("vms group %s flist: %s is invalid, failed to download flist", name, flist)
		}
		defer response.Body.Close()
	}
	return nil
}
