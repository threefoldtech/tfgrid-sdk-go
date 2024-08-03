package config

import "errors"

var (
	ErrVersionNotSet               = errors.New("version not set")
	ErrNetworkTypeNotSet           = errors.New("network type not set")
	ErrServiceFlistNotSet          = errors.New("service flist not set")
	ErrServiceCPUResourceNotSet    = errors.New("service cpu resource not set")
	ErrServiceMemoryResourceNotSet = errors.New("service memory resource not set")
	ErrStorageTypeNotSet           = errors.New("storage type not set")
	ErrStorageSizeNotSet           = errors.New("storage size not set")
)
