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
	"github.com/advancedhosting/csi-advancedhosting/cloud"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
	"os"
	"path/filepath"
)

const (
	DefaultFSType     = "ext4"
	MaxVolumesPerNode = 8
	InstancePath      = "/var/lib/cloud/instance"
)

type nodeService struct {
	client        *ah.APIClient
	mounter       Mounter
	cloudServerID string
	datacenterID  string
}

func cloudServerID() (string, error) {
	var csID string
	csID, err := cloudServerIDFromMetadata()
	if err != nil {
		klog.Warning("error getting cloud server id from metadata: ", err, ".Trying to get id from local")
		csID, err = cloudServerIDFromLocal()
		if err != nil {
			return "", err
		}
	}
	return csID, nil
}

func cloudServerIDFromMetadata() (string, error) {
	metadata, err := cloud.NewMetadata()
	if err != nil {
		return "", fmt.Errorf("an error occurred while creating metadata client")
	}
	csID, err := metadata.CloudServerID()
	if err != nil || csID == "" {
		return "", fmt.Errorf("an error occurred while getting cloud server ID %s: %v", csID, err)
	}
	return csID, nil
}

func cloudServerIDFromLocal() (string, error) {
	realPath, err := os.Readlink(InstancePath)
	if err != nil {
		return "", err
	}
	_, csID := filepath.Split(realPath)
	if csID == "" {
		return "", fmt.Errorf("an error occurred while getting cloud server id from local: Empty symlink")
	}
	return csID, nil
}

func NewNodeService(client *ah.APIClient) (*nodeService, error) {
	csID, err := cloudServerID()
	if err != nil || csID == "" {
		return nil, fmt.Errorf("an error occurred while getting cloud server ID %s: %v", csID, err)
	}

	cloudServer, err := client.Instances.Get(context.Background(), csID)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while getting cloud server info: %s", err)
	}
	return &nodeService{
		client:        client,
		mounter:       NewLinuxMounter(),
		cloudServerID: csID,
		datacenterID:  cloudServer.Datacenter.ID,
	}, nil
}

func (n *nodeService) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	klog.V(4).Infof("NodeStageVolume: %+v", *req)
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID must be provided")
	}

	targetPath := req.GetStagingTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Staging target must be provided")
	}

	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability must be provided")
	}

	if !isValidVolumeCapability(volCap) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	if volCap.GetBlock() != nil {
		return &csi.NodeStageVolumeResponse{}, nil
	}

	mount := volCap.GetMount()
	if mount == nil {
		return nil, status.Error(codes.InvalidArgument, "Mount volume capability must be provided")
	}

	fsType := mount.GetFsType()
	if fsType == "" {
		fsType = DefaultFSType
	}
	volume, err := n.client.Volumes.Get(ctx, volumeID)
	if err != nil {
		if err == ah.ErrResourceNotFound {
			return nil, status.Errorf(codes.NotFound, "Volume %s not found", volumeID)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	devicePath, err := n.mounter.DevicePath(volume.Number)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	mounted, err := n.mounter.IsDeviceMountedToTarget(devicePath, targetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if mounted {
		klog.V(4).Infof("device %s already mounted to %s", devicePath, targetPath)
		return &csi.NodeStageVolumeResponse{}, nil
	}

	exists, err := n.mounter.IsPathExists(targetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		if err := n.mounter.CreateDir(targetPath); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	if err := n.mounter.FormatAndMount(devicePath, targetPath, fsType, mount.MountFlags); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

func (n *nodeService) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	klog.V(4).Infof("NodeUnstageVolume: %+v", *req)
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID must be provided")
	}

	targetPath := req.GetStagingTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Staging target must be provided")
	}

	volume, err := n.client.Volumes.Get(ctx, volumeID)
	if err != nil {
		if err == ah.ErrResourceNotFound {
			return nil, status.Errorf(codes.NotFound, "Volume %s not found", volumeID)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	devicePath, err := n.mounter.DevicePath(volume.Number)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	mounted, err := n.mounter.IsDeviceMountedToTarget(devicePath, targetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !mounted {
		klog.V(4).Infof("device %s not mounted to %s", devicePath, targetPath)
		return &csi.NodeUnstageVolumeResponse{}, nil
	}

	if err := n.mounter.Unmount(targetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (n *nodeService) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	klog.V(4).Infof("NodePublishVolume: %+v", *req)
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID must be provided")
	}

	stagingTargetPath := req.GetStagingTargetPath()
	if stagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Staging target must be provided")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path must be provided")
	}

	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability must be provided")
	}

	if !isValidVolumeCapability(volCap) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	volume, err := n.client.Volumes.Get(ctx, volumeID)
	if err != nil {
		if err == ah.ErrResourceNotFound {
			return nil, status.Errorf(codes.NotFound, "Volume %s not found", volumeID)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	options := []string{"bind"}
	if req.GetReadonly() {
		options = append(options, "ro")
	}

	switch mode := volCap.GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		devicePath, err := n.mounter.DevicePath(volume.Number)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		if err := n.nodePublishBlockVolume(devicePath, targetPath, options); err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	case *csi.VolumeCapability_Mount:
		options = append(options, mode.Mount.MountFlags...)
		fsType := mode.Mount.GetFsType()
		if fsType == "" {
			fsType = DefaultFSType
		}
		if err := n.nodeMountFSVolume(stagingTargetPath, targetPath, fsType, options); err != nil {
			return nil, err
		}
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (n *nodeService) nodePublishBlockVolume(source, target string, options []string) error {
	if err := n.mounter.CreateDir(filepath.Dir(target)); err != nil {
		return err
	}
	if err := n.mounter.CreateFile(target); err != nil {
		return err
	}
	if err := n.mounter.Mount(source, target, "", options); err != nil {
		return err
	}
	return nil
}

func (n *nodeService) nodeMountFSVolume(source, target, fsType string, options []string) error {
	if err := n.mounter.CreateDir(target); err != nil {
		return err
	}
	if err := n.mounter.Mount(source, target, fsType, options); err != nil {
		return err
	}
	return nil
}

func (n *nodeService) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.V(4).Infof("NodeUnpublishVolume: %+v", *req)

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID must be provided")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path must be provided")
	}
	if _, err := n.client.Volumes.Get(ctx, volumeID); err != nil {
		if err == ah.ErrResourceNotFound {
			return nil, status.Errorf(codes.NotFound, "Volume %s not found", volumeID)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := n.mounter.Unmount(targetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (n *nodeService) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	klog.V(4).Infof("NodeGetVolumeStats: %+v", *req)

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID must be provided")
	}

	volumePath := req.GetVolumePath()
	if volumePath == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume path must be provided")
	}

	if _, err := n.client.Volumes.Get(ctx, volumeID); err != nil {
		if err == ah.ErrResourceNotFound {
			return nil, status.Errorf(codes.NotFound, "Volume %s not found", volumeID)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	exists, err := n.mounter.IsPathExists(volumePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "Path %s not found", volumePath)
	}

	isBlock, err := n.mounter.IsBlockDevice(volumePath)

	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if isBlock {
		size, err := n.mounter.BlockSize(volumePath)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		return &csi.NodeGetVolumeStatsResponse{
			Usage: []*csi.VolumeUsage{
				{
					Unit:  csi.VolumeUsage_BYTES,
					Total: size,
				},
			},
		}, nil
	}

	bAvailable, bTotal, bUsed, err := n.mounter.BytesFSMetrics(volumePath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	iNodeAvailable, iNodeTotal, iNodeUsed, err := n.mounter.INodeFSMetrics(volumePath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			{
				Unit:      csi.VolumeUsage_BYTES,
				Available: bAvailable,
				Total:     bTotal,
				Used:      bUsed,
			},
			{
				Unit:      csi.VolumeUsage_INODES,
				Available: iNodeAvailable,
				Total:     iNodeTotal,
				Used:      iNodeUsed,
			},
		},
	}, nil
}

func (n *nodeService) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	klog.V(4).Infof("NodeExpandVolume: %+v", *req)

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID must be provided")
	}
	if _, err := n.client.Volumes.Get(ctx, volumeID); err != nil {
		if err == ah.ErrResourceNotFound {
			return nil, status.Errorf(codes.NotFound, "Volume %s not found", volumeID)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	volumePath := req.GetVolumePath()
	if volumePath == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume path must be provided")
	}

	volCap := req.GetVolumeCapability()
	if volCap != nil {
		if !isValidVolumeCapability(volCap) {
			return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
		}

		if blockCap := volCap.GetBlock(); blockCap != nil {
			klog.V(4).Infof("NodeExpandVolume: skip for block device")
			return &csi.NodeExpandVolumeResponse{}, nil
		}
	}

	devicePath, err := n.mounter.DeviceNameFromMount(volumePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if devicePath == "" {
		return nil, status.Errorf(codes.NotFound, "NodeExpandVolume: devicePath %s not found", volumePath)
	}

	if err := n.mounter.Resize(devicePath, volumePath); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &csi.NodeExpandVolumeResponse{}, nil
}

func (n *nodeService) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	klog.V(4).Infof("NodeGetCapabilities: %+v", *req)
	nodCaps := []*csi.NodeServiceCapability{
		{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
				},
			},
		},
		{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
				},
			},
		},
		{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
				},
			},
		},
	}

	return &csi.NodeGetCapabilitiesResponse{Capabilities: nodCaps}, nil
}

func (n *nodeService) NodeGetInfo(context.Context, *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId:            n.cloudServerID,
		MaxVolumesPerNode: MaxVolumesPerNode,
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{
				TopologySegmentDatacenter: n.datacenterID,
			},
		},
	}, nil
}
