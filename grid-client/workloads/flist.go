// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"
)

// FlistChecksumURL returns flist check sum url format
func FlistChecksumURL(url string) string {
	return fmt.Sprintf("%s.md5", url)
}

// GetFlistChecksum gets flist check sum url
func GetFlistChecksum(url string) (string, error) {
	cl := &http.Client{
		Timeout: 10 * time.Second,
	}
	response, err := cl.Get(FlistChecksumURL(url))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	hash, err := io.ReadAll(response.Body)
	return strings.TrimSpace(string(hash)), err
}

func ValidateFlist(flistUrl string) error {
	flistExt := path.Ext(flistUrl)
	if flistExt != ".fl" && flistExt != ".flist" {
		return fmt.Errorf("flist: '%s' is invalid, should have a valid flist extension", flistUrl)
	}

	cl := &http.Client{
		Timeout: 10 * time.Second,
	}

	response, err := cl.Head(flistUrl)
	if err != nil || response.StatusCode != http.StatusOK {
		return fmt.Errorf("flist: '%s' is invalid, failed to download flist", flistUrl)
	}
	defer response.Body.Close()

	return nil
}
