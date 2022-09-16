/*
Copyright 2021 Advanced Hosting

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"context"
	"fmt"
	"github.com/advancedhosting/advancedhosting-api-go/ah"
	"github.com/google/uuid"
	"github.com/kubernetes-csi/csi-test/v3/pkg/sanity"
	"os"
	"path/filepath"
	"testing"
)

func TestSanity(t *testing.T) {
	dir, err := os.MkdirTemp("", "sanity-csi")
	if err != nil {
		t.Fatalf("error creating directory %v", err)
	}
	defer os.RemoveAll(dir)

	targetPath := filepath.Join(dir, "mount")
	stagingPath := filepath.Join(dir, "staging")
	endpoint := "unix://" + filepath.Join(dir, "csi.sock")
	config := sanity.NewTestConfig()
	config.TargetPath = targetPath
	config.StagingPath = stagingPath
	config.Address = endpoint
	config.CreateTargetDir = createDir
	config.CreateStagingDir = createDir

	client := &ah.APIClient{
		Instances: &fakeInstancesAPI{},
		Volumes:   &fakeVolumesAPI{},
	}

	controllerSvc := &controllerService{
		client:               client,
		defaultVolumeProduct: "default-volume-product-slug",
	}
	nodeSvc := &nodeService{
		client:        client,
		mounter:       &fakeLinuxMounter{},
		cloudServerID: "test-cloud-server-id",
		datacenterID:  "test-datacenter-id",
	}

	driver := &Driver{
		endpoint:          endpoint,
		controllerService: controllerSvc,
		nodeService:       nodeSvc,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("recover: %v", r)
		}
	}()
	go func() {
		if err := driver.Run(); err != nil {
			panic(fmt.Sprintf("%v", err))
		}
	}()

	sanity.Test(t, config)
}

func createDir(targetPath string) (string, error) {
	if err := os.MkdirAll(targetPath, 0300); err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
	}
	return targetPath, nil
}

type fakeInstancesAPI struct {
}

func (is *fakeInstancesAPI) Create(ctx context.Context, createRequest *ah.InstanceCreateRequest) (*ah.Instance, error) {
	panic("not implemented")
}

func (is *fakeInstancesAPI) Rename(ctx context.Context, instanceID, name string) (*ah.Instance, error) {
	panic("not implemented")
}

func (is *fakeInstancesAPI) Upgrade(ctx context.Context, instanceID string, request *ah.InstanceUpgradeRequest) error {
	panic("not implemented")
}

func (is *fakeInstancesAPI) Shutdown(ctx context.Context, instanceID string) error {
	panic("not implemented")
}

func (is *fakeInstancesAPI) PowerOff(ctx context.Context, instanceID string) error {
	panic("not implemented")
}

func (is *fakeInstancesAPI) Destroy(ctx context.Context, instanceID string) error {
	panic("not implemented")
}

// List returns all available instances
func (is *fakeInstancesAPI) List(ctx context.Context, options *ah.ListOptions) ([]ah.Instance, *ah.Meta, error) {
	panic("not implemented")
}

// Get returns all instance by instanceID
func (is *fakeInstancesAPI) Get(ctx context.Context, instanceID string) (*ah.Instance, error) {
	panic("not implemented")
}

func (is *fakeInstancesAPI) SetPrimaryIP(ctx context.Context, instanceID, ipAssignmentID string) (*ah.Action, error) {
	panic("not implemented")
}

func (is *fakeInstancesAPI) ActionInfo(ctx context.Context, instanceID, actionID string) (*ah.InstanceAction, error) {
	return &ah.InstanceAction{Action: &ah.Action{State: "success"}}, nil
}

func (is *fakeInstancesAPI) Actions(ctx context.Context, instanceID string) ([]ah.InstanceAction, error) {
	panic("not implemented")
}

func (is *fakeInstancesAPI) AttachVolume(ctx context.Context, instanceID, volumeID string) (*ah.Action, error) {
	if instanceID != "test-cloud-server-id" {
		return nil, ah.ErrResourceNotFound
	}
	return &ah.Action{}, nil
}

func (is *fakeInstancesAPI) DetachVolume(ctx context.Context, instanceID, volumeID string) (*ah.Action, error) {
	return &ah.Action{}, nil
}

func (is *fakeInstancesAPI) AvailableVolumes(ctx context.Context, instanceID string, options *ah.ListOptions) ([]ah.Volume, *ah.Meta, error) {
	panic("not implemented")
}

func (is *fakeInstancesAPI) CreateBackup(ctx context.Context, instanceID, note string) (*ah.InstanceAction, error) {
	panic("not implemented")
}

type fakeVolumesAPI struct {
	volumes []ah.Volume
}

func (vs *fakeVolumesAPI) List(ctx context.Context, options *ah.ListOptions) ([]ah.Volume, *ah.Meta, error) {
	return vs.volumes, nil, nil
}

func (vs *fakeVolumesAPI) Get(ctx context.Context, volumeID string) (*ah.Volume, error) {
	for _, volume := range vs.volumes {
		if volume.ID == volumeID {
			return &volume, nil
		}
	}
	return nil, ah.ErrResourceNotFound
}

func (vs *fakeVolumesAPI) Create(ctx context.Context, createRequest *ah.VolumeCreateRequest) (*ah.Volume, error) {
	volume := &ah.Volume{
		ID:    uuid.New().String(),
		Size:  createRequest.Size,
		State: "ready",
	}
	volume.VolumePool = (*struct {
		Name             string   `json:"name,omitempty"`
		DatacenterIDs    []string `json:"datacenter_ids,omitempty"`
		ReplicationLevel int      `json:"replication_level,omitempty"`
	})(&struct {
		Name             string
		DatacenterIDs    []string
		ReplicationLevel int
	}{DatacenterIDs: []string{"test-datacenter-id"}})
	vs.volumes = append(vs.volumes, *volume)
	return volume, nil

}

func (vs *fakeVolumesAPI) Update(ctx context.Context, volumeID string, request *ah.VolumeUpdateRequest) (*ah.Volume, error) {
	panic("not implemented")
}

func (vs *fakeVolumesAPI) Copy(ctx context.Context, volumeID string, request *ah.VolumeCopyActionRequest) (*ah.VolumeAction, error) {
	panic("not implemented")
}

func (vs *fakeVolumesAPI) Resize(ctx context.Context, volumeID string, size int) (*ah.Action, error) {
	return &ah.Action{}, nil
}

func (vs *fakeVolumesAPI) ActionInfo(ctx context.Context, volumeID, actionID string) (*ah.VolumeAction, error) {
	return &ah.VolumeAction{Action: &ah.Action{
		State: "success",
	}}, nil
}

func (vs *fakeVolumesAPI) Actions(ctx context.Context, volumeID string) ([]ah.VolumeAction, error) {
	return []ah.VolumeAction{{}}, nil
}

func (vs *fakeVolumesAPI) Delete(ctx context.Context, volumeID string) error {
	for index, volume := range vs.volumes {
		if volume.ID == volumeID {
			vs.volumes = append(vs.volumes[:index], vs.volumes[index+1:]...)
			return nil
		}
	}
	return ah.ErrResourceNotFound
}

type fakeLinuxMounter struct {
}

var _ Mounter = (*fakeLinuxMounter)(nil)

func (m *fakeLinuxMounter) DevicePath(volumeNumber string) (string, error) {
	return "", nil
}

func (m *fakeLinuxMounter) IsPathExists(path string) (bool, error) {
	if path == "some/path" {
		return false, nil
	}
	return true, nil
}

func (m *fakeLinuxMounter) FormatAndMount(source, target, fsType string, options []string) error {
	return nil
}

func (m *fakeLinuxMounter) Mount(source, target, fsType string, options []string) error {
	return nil
}

func (m *fakeLinuxMounter) CreateDir(path string) error {
	return nil
}

func (m *fakeLinuxMounter) CreateFile(path string) error {
	return nil
}

func (m *fakeLinuxMounter) IsDeviceMountedToTarget(devicePath, targetPath string) (bool, error) {
	return true, nil
}

func (m *fakeLinuxMounter) DeviceNameFromMount(mountPath string) (string, error) {
	return "", nil
}

func (m *fakeLinuxMounter) Unmount(target string) error {
	return nil
}

func (m *fakeLinuxMounter) IsBlockDevice(path string) (bool, error) {
	return true, nil
}

func (m *fakeLinuxMounter) BlockSize(path string) (int64, error) {
	return int64(0), nil
}

func (m *fakeLinuxMounter) BytesFSMetrics(path string) (available, total, used int64, err error) {
	available = int64(0)
	total = int64(0)
	used = int64(0)
	err = nil
	return
}

func (m *fakeLinuxMounter) INodeFSMetrics(path string) (available, total, used int64, err error) {
	available = int64(0)
	total = int64(0)
	used = int64(0)
	err = nil
	return
}

func (m *fakeLinuxMounter) Resize(devicePath, deviceMountPath string) error {
	return nil
}
