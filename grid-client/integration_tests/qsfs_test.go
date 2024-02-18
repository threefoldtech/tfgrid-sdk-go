// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

const (
	DataZDBNum = 4
	MetaZDBNum = 4
	zdbSize    = 1
)

func TestQSFSDeployment(t *testing.T) {
	tfPluginClient, err := setup()
	if !assert.NoError(t, err) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	publicKey, privateKey, err := GenerateSSHKeyPair()
	if !assert.NoError(t, err) {
		return
	}

	zdbSizes := make([]uint64, 0)
	for i := 0; i < DataZDBNum+MetaZDBNum; i++ {
		zdbSizes = append(zdbSizes, zdbSize)
	}

	nodes, err := deployer.FilterNodes(ctx, tfPluginClient, nodeFilter, nil, zdbSizes, []uint64{minRootfs})
	if err != nil {
		t.Skip("no available nodes found")
	}

	nodeID := uint32(nodes[0].NodeID)

	network := workloads.ZNet{
		Name:        generateRandString(10),
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: gridtypes.NewIPNet(net.IPNet{
			IP:   net.IPv4(10, 20, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}),
		AddWGAccess: false,
	}

	dataZDBs := []workloads.ZDB{}
	metaZDBs := []workloads.ZDB{}
	for i := 1; i <= DataZDBNum; i++ {
		zdb := workloads.ZDB{
			Name:        "qsfsDataZdb" + strconv.Itoa(i),
			Password:    "password",
			Public:      true,
			Size:        zdbSize,
			Description: "zdb for testing",
			Mode:        zos.ZDBModeSeq,
		}
		dataZDBs = append(dataZDBs, zdb)
	}

	for i := 1; i <= MetaZDBNum; i++ {
		zdb := workloads.ZDB{
			Name:        "qsfsMetaZdb" + strconv.Itoa(i),
			Password:    "password",
			Public:      true,
			Size:        zdbSize,
			Description: "zdb for testing",
			Mode:        zos.ZDBModeUser,
		}
		metaZDBs = append(metaZDBs, zdb)
	}

	dl1 := workloads.NewDeployment(generateRandString(10), nodeID, "", nil, "", nil, append(dataZDBs, metaZDBs...), nil, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl1)
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl1)
		assert.NoError(t, err)
	}()

	// result zdbs
	resDataZDBs := []workloads.ZDB{}
	resMetaZDBs := []workloads.ZDB{}
	for i := 1; i <= DataZDBNum; i++ {
		res, err := tfPluginClient.State.LoadZdbFromGrid(ctx, nodeID, "qsfsDataZdb"+strconv.Itoa(i), dl1.Name)
		if !assert.NoError(t, err) || !assert.NotEmpty(t, res) {
			return
		}
		resDataZDBs = append(resDataZDBs, res)
	}
	for i := 1; i <= MetaZDBNum; i++ {
		res, err := tfPluginClient.State.LoadZdbFromGrid(ctx, nodeID, "qsfsMetaZdb"+strconv.Itoa(i), dl1.Name)
		if !assert.NoError(t, err) || !assert.NotEmpty(t, res) {
			return
		}
		resMetaZDBs = append(resMetaZDBs, res)
	}

	// backends
	dataBackends := []workloads.Backend{}
	metaBackends := []workloads.Backend{}
	for i := 0; i < DataZDBNum; i++ {
		dataBackends = append(dataBackends, workloads.Backend{
			Address:   "[" + resDataZDBs[i].IPs[1] + "]" + ":" + fmt.Sprint(resDataZDBs[i].Port),
			Namespace: resDataZDBs[i].Namespace,
			Password:  resDataZDBs[i].Password,
		})
	}
	for i := 0; i < MetaZDBNum; i++ {
		metaBackends = append(metaBackends, workloads.Backend{
			Address:   "[" + resMetaZDBs[i].IPs[1] + "]" + ":" + fmt.Sprint(resMetaZDBs[i].Port),
			Namespace: resMetaZDBs[i].Namespace,
			Password:  resMetaZDBs[i].Password,
		})
	}

	qsfs := workloads.QSFS{
		Name:                 "qsfs",
		Description:          "qsfs for testing",
		Cache:                1024,
		MinimalShards:        2,
		ExpectedShards:       4,
		RedundantGroups:      0,
		RedundantNodes:       0,
		MaxZDBDataDirSize:    512,
		EncryptionAlgorithm:  "AES",
		EncryptionKey:        "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af",
		CompressionAlgorithm: "snappy",
		Groups:               workloads.Groups{{Backends: dataBackends}},
		Metadata: workloads.Metadata{
			Type:                "zdb",
			Prefix:              "test",
			EncryptionAlgorithm: "AES",
			EncryptionKey:       "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af",
			Backends:            metaBackends,
		},
	}

	vm := workloads.VM{
		Name:       "vm",
		Flist:      "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		CPU:        2,
		Planetary:  true,
		Memory:     1024,
		Entrypoint: "/sbin/zinit init",
		EnvVars: map[string]string{
			"SSH_KEY": publicKey,
		},
		Mounts: []workloads.Mount{
			{DiskName: qsfs.Name, MountPoint: "/qsfs"},
		},
		NetworkName: network.Name,
	}

	err = tfPluginClient.NetworkDeployer.Deploy(ctx, &network)
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.NetworkDeployer.Cancel(ctx, &network)
		assert.NoError(t, err)
	}()

	dl2 := workloads.NewDeployment(generateRandString(10), nodeID, "", nil, network.Name, nil, append(dataZDBs, metaZDBs...), []workloads.VM{vm}, []workloads.QSFS{qsfs})
	err = tfPluginClient.DeploymentDeployer.Deploy(ctx, &dl2)
	if !assert.NoError(t, err) {
		return
	}

	defer func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(ctx, &dl2)
		assert.NoError(t, err)
	}()

	resVM, err := tfPluginClient.State.LoadVMFromGrid(ctx, nodeID, vm.Name, dl2.Name)
	if !assert.NoError(t, err) {
		return
	}

	resQSFS, err := tfPluginClient.State.LoadQSFSFromGrid(ctx, nodeID, qsfs.Name, dl2.Name)
	if !assert.NoError(t, err) || !assert.NotEmpty(t, resQSFS.MetricsEndpoint) {
		return
	}

	// Check that the outputs not empty
	metrics := resQSFS.MetricsEndpoint
	if !assert.NotEmpty(t, metrics) {
		return
	}

	planetaryIP := resVM.PlanetaryIP
	if !assert.NotEmpty(t, yggIP) {
		return
	}

	// get metrics
	cmd := exec.Command("curl", metrics)
	output, err := cmd.Output()
	if !assert.NoError(t, err) || !assert.Contains(t, string(output), "fs_syscalls{syscall=\"create\"} 0") {
		return
	}

	// try write to a file in mounted disk
	_, err = RemoteRun("root", planetaryIP, "cd /qsfs && echo hamadatext >> hamadafile", privateKey)
	if !assert.NoError(t, err) {
		return
	}

	// get metrics after write
	cmd = exec.Command("curl", metrics)
	output, err = cmd.Output()
	assert.NoError(t, err)
	assert.Contains(t, string(output), "fs_syscalls{syscall=\"create\"} 1")

	resQSFS.MetricsEndpoint = ""
	assert.Equal(t, qsfs, resQSFS)
}
