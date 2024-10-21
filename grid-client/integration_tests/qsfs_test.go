// Package integration for integration tests
package integration

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/deployer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

const (
	DataZDBNum = 4
	MetaZDBNum = 4
	zdbSize    = 1
)

func TestQSFSDeployment(t *testing.T) {
	t.Skipf("related issue: https://github.com/threefoldtech/zos/issues/2403")
	tfPluginClient, err := setup()
	if err != nil {
		t.Skipf("plugin creation failed: %v", err)
	}

	if tfPluginClient.Network == "test" {
		t.Skipf("https://github.com/threefoldtech/tfgrid-sdk-go/issues/1111")
	}

	publicKey, privateKey, err := GenerateSSHKeyPair()
	require.NoError(t, err)

	zdbSizes := make([]uint64, 0)
	for i := 0; i < DataZDBNum+MetaZDBNum; i++ {
		zdbSizes = append(zdbSizes, zdbSize)
	}

	nodes, err := deployer.FilterNodes(
		context.Background(),
		tfPluginClient,
		generateNodeFilter(WithFreeHRU(DataZDBNum+MetaZDBNum)),
		nil,
		zdbSizes,
		nil,
		1,
	)
	if err != nil {
		t.Skipf("no available nodes found: %v", err)
	}

	nodeID := uint32(nodes[0].NodeID)

	network, err := generateBasicNetwork([]uint32{nodeID})
	require.NoError(t, err)

	dataZDBs := []workloads.ZDB{}
	metaZDBs := []workloads.ZDB{}
	for i := 1; i <= DataZDBNum; i++ {
		zdb := workloads.ZDB{
			Name:        "qsfsDataZdb" + strconv.Itoa(i),
			Password:    "password",
			Public:      true,
			SizeGB:      zdbSize,
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
			SizeGB:      zdbSize,
			Description: "zdb for testing",
			Mode:        zos.ZDBModeUser,
		}
		metaZDBs = append(metaZDBs, zdb)
	}

	dl1 := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, "", nil, append(dataZDBs, metaZDBs...), nil, nil, nil, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &dl1)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(context.Background(), &dl1)
		require.NoError(t, err)
	})

	// result zdbs
	resDataZDBs := []workloads.ZDB{}
	resMetaZDBs := []workloads.ZDB{}
	for i := 1; i <= DataZDBNum; i++ {
		res, err := tfPluginClient.State.LoadZdbFromGrid(context.Background(), nodeID, "qsfsDataZdb"+strconv.Itoa(i), dl1.Name)
		require.NoError(t, err)
		require.NotEmpty(t, res)
		resDataZDBs = append(resDataZDBs, res)
	}

	for i := 1; i <= MetaZDBNum; i++ {
		res, err := tfPluginClient.State.LoadZdbFromGrid(context.Background(), nodeID, "qsfsMetaZdb"+strconv.Itoa(i), dl1.Name)
		require.NoError(t, err)
		require.NotEmpty(t, res)
		resMetaZDBs = append(resMetaZDBs, res)
	}

	// backends
	dataBackends := []workloads.Backend{}
	metaBackends := []workloads.Backend{}
	for i := 0; i < DataZDBNum; i++ {
		dataBackends = append(dataBackends, workloads.Backend{
			Address:   "[" + resDataZDBs[i].IPs[2] + "]" + ":" + fmt.Sprint(resDataZDBs[i].Port),
			Namespace: resDataZDBs[i].Namespace,
			Password:  resDataZDBs[i].Password,
		})
	}

	for i := 0; i < MetaZDBNum; i++ {
		metaBackends = append(metaBackends, workloads.Backend{
			Address:   "[" + resMetaZDBs[i].IPs[2] + "]" + ":" + fmt.Sprint(resMetaZDBs[i].Port),
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

	vm, err := generateBasicVM("vm", nodeID, network.Name, publicKey)
	require.NoError(t, err)

	vm.Mounts = []workloads.Mount{
		{Name: qsfs.Name, MountPoint: "/qsfs"},
	}

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &network)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.NetworkDeployer.Cancel(context.Background(), &network)
		require.NoError(t, err)
	})

	dl2 := workloads.NewDeployment(fmt.Sprintf("dl_%s", generateRandString(10)), nodeID, "", nil, network.Name, nil, append(dataZDBs, metaZDBs...), []workloads.VM{vm}, nil, []workloads.QSFS{qsfs}, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &dl2)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = tfPluginClient.DeploymentDeployer.Cancel(context.Background(), &dl2)
		require.NoError(t, err)
	})

	resVM, err := tfPluginClient.State.LoadVMFromGrid(context.Background(), nodeID, vm.Name, dl2.Name)
	require.NoError(t, err)

	resQSFS, err := tfPluginClient.State.LoadQSFSFromGrid(context.Background(), nodeID, qsfs.Name, dl2.Name)
	require.NoError(t, err)
	require.NotEmpty(t, resQSFS.MetricsEndpoint)

	// Check that the outputs not empty
	metrics := resQSFS.MetricsEndpoint
	require.NotEmpty(t, metrics)

	myceliimIP := resVM.MyceliumIP
	require.NotEmpty(t, myceliimIP)

	// get metrics
	cmd := exec.Command("curl", metrics)
	output, err := cmd.Output()
	require.NoError(t, err)
	require.Contains(t, string(output), "fs_syscalls{syscall=\"create\"} 0")

	// try write to a file in mounted disk
	_, err = RemoteRun("root", myceliimIP, "cd /qsfs && echo hamadatext >> hamadafile", privateKey)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	// get metrics after write
	cmd = exec.Command("curl", metrics)
	output, err = cmd.Output()
	require.NoError(t, err)
	require.Contains(t, string(output), "fs_syscalls{syscall=\"create\"} 1")

	resQSFS.MetricsEndpoint = ""
	require.Equal(t, qsfs, resQSFS)
}
