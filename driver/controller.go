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
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
	"strings"
)

const (
	gb                   = 1 << (10 * 3)
	MinVolumeSize        = 10 * gb
	DefaultVolumeSize    = MinVolumeSize
	VolumeProductSlugKey = "product-slug"
)

func isValidVolumeCapability(volCap *csi.VolumeCapability) bool {
	switch mode := volCap.GetAccessMode().GetMode(); mode {
	case csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER:
		return true
	default:
		return false
	}
}

func volumeSize(capRange *csi.CapacityRange) (int64, error) {
	if capRange == nil {
		return DefaultVolumeSize / gb, nil
	}

	minBytes := capRange.GetRequiredBytes()
	maxBytes := capRange.GetLimitBytes()
	if minBytes <= 0 && maxBytes <= 0 {
		return DefaultVolumeSize / gb, nil
	}

	minSize := minBytes
	if minSize < MinVolumeSize {
		minSize = MinVolumeSize
	}

	if maxBytes > 0 && minSize > maxBytes {
		return 0, fmt.Errorf("volume size exceeds the limit")
	}

	return minSize / gb, nil
}

type controllerService struct {
	client               *ah.APIClient
	defaultVolumeProduct string
	clusterID            string
}

func NewControllerService(client *ah.APIClient, clusterID string) *controllerService {
	return &controllerService{client: client, clusterID: clusterID}
}

func (c *controllerService) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	klog.V(4).Infof("ControllerCreateVolume: %+v", *req)

	volName := req.GetName()
	if volName == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume name not provided")
	}
	if c.clusterID != "" {
		volName = fmt.Sprintf("kube-%s-%s", c.clusterID, volName)
	}

	var productSlug string
	if slug, ok := req.Parameters[VolumeProductSlugKey]; ok {
		productSlug = slug
	} else {
		productSlug = c.defaultVolumeProduct
	}
	if productSlug == "" {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("%s param must be provided", VolumeProductSlugKey))
	}

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}
	for i, volCap := range volCaps {
		if !isValidVolumeCapability(volCap) {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Volume capability %d not provided", i))
		}
	}

	size, err := volumeSize(req.GetCapacityRange())
	if err != nil {
		return nil, status.Error(codes.OutOfRange, "Volume size exceeds the limit")
	}

	filter := &ah.EqFilter{
		Keys:  []string{"name"},
		Value: volName,
	}

	options := &ah.ListOptions{
		Filters: []ah.FilterInterface{filter},
	}
	volumes, _, err := c.client.Volumes.List(ctx, options)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(volumes) > 0 {
		if len(volumes) > 1 {
			return nil, status.Error(codes.AlreadyExists, fmt.Sprintf("volume %s already exists", volName))
		}
		volume := volumes[0]
		if volume.Size != int(size) {
			return nil, status.Error(codes.AlreadyExists, fmt.Sprintf("invalid requested size: %d", size))
		}

		return c.createVolumeResponse(&volume), nil
	}

	createRequest := &ah.VolumeCreateRequest{
		Name:        volName,
		Size:        int(size),
		ProductSlug: productSlug,
	}

	volume, err := c.client.Volumes.Create(ctx, createRequest)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := c.waitForVolumeState(ctx, volume.ID, "ready"); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return c.createVolumeResponse(volume), nil
}

func (c *controllerService) createVolumeResponse(volume *ah.Volume) *csi.CreateVolumeResponse {
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volume.ID,
			CapacityBytes: int64(volume.Size) * gb,
			AccessibleTopology: []*csi.Topology{
				{
					Segments: map[string]string{
						TopologySegmentDatacenter: volume.VolumePool.DatacenterIDs[0],
					},
				},
			},
		},
	}
}

func (c *controllerService) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.V(4).Infof("ControllerDeleteVolume: %+v", *req)
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID must be provided")
	}

	err := c.client.Volumes.Delete(ctx, req.VolumeId)
	if err != nil {
		if err == ah.ErrResourceNotFound {
			return &csi.DeleteVolumeResponse{}, nil
		}
		if strings.Contains(err.Error(), "performing another action: Destroy") {
			return nil, status.Errorf(codes.Aborted, err.Error())
		}
		return nil, err
	}
	return &csi.DeleteVolumeResponse{}, nil
}

func (c *controllerService) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	klog.V(4).Infof("ControllerPublishVolume: %+v", *req)

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID must be provided")
	}

	nodeID := req.GetNodeId()
	if nodeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Node ID must be provided")
	}

	volCap := req.GetVolumeCapability()

	if !isValidVolumeCapability(volCap) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}

	volume, err := c.client.Volumes.Get(ctx, volumeID)
	if err != nil {
		if err == ah.ErrResourceNotFound {
			return nil, status.Errorf(codes.NotFound, "Volume %s not found", volumeID)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if volume.Instance != nil {
		if volume.Instance.ID != nodeID {
			return nil, status.Errorf(
				codes.FailedPrecondition, "Volume %s is already attached to the another node %s", volume.ID, volume.Instance.ID)
		}
		return &csi.ControllerPublishVolumeResponse{}, nil
	}

	action, err := c.client.Instances.AttachVolume(ctx, nodeID, volumeID)
	if err != nil {
		if err == ah.ErrResourceNotFound {
			return nil, status.Errorf(codes.NotFound, "Cloud Server %s not found", nodeID)
		}
		if strings.Contains(err.Error(), "Reached maximum volumes number") {
			return nil, status.Errorf(codes.ResourceExhausted, err.Error())
		}
		if strings.Contains(err.Error(), "performing another action: Attach volume") {
			return nil, status.Errorf(codes.Aborted, err.Error())
		}
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if err := c.waitForCloudServerActionState(ctx, nodeID, action.ID, "success"); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	klog.V(4).Infof("ControllerPublishVolume: volume %s attached to node %s ", volumeID, nodeID)

	return &csi.ControllerPublishVolumeResponse{}, nil
}

func (c *controllerService) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	klog.V(4).Infof("ControllerUnpublishVolume: %+v", *req)

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID must be provided")
	}
	if _, err := c.client.Volumes.Get(ctx, volumeID); err != nil {
		if err == ah.ErrResourceNotFound {
			return &csi.ControllerUnpublishVolumeResponse{}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	nodeID := req.GetNodeId()
	if nodeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Node ID must be provided")
	}

	action, err := c.client.Instances.DetachVolume(ctx, nodeID, volumeID)
	if err != nil {
		if err == ah.ErrResourceNotFound || strings.Contains(err.Error(), "not attached") {
			return &csi.ControllerUnpublishVolumeResponse{}, nil
		}
		if strings.Contains(err.Error(), "performing another action: Detach volume") {
			return nil, status.Errorf(codes.Aborted, err.Error())
		}
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if err := c.waitForCloudServerActionState(ctx, nodeID, action.ID, "success"); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (c *controllerService) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	klog.V(4).Infof("ValidateVolumeCapabilities: %+v", *req)

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID must be provided")
	}
	if _, err := c.client.Volumes.Get(ctx, volumeID); err != nil {
		if err == ah.ErrResourceNotFound {
			return nil, status.Errorf(codes.NotFound, "Volume %s not found", volumeID)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	isValidVolumeCapabilities := func(volCaps []*csi.VolumeCapability) bool {
		for _, volCap := range volCaps {
			if !isValidVolumeCapability(volCap) {
				return false
			}
		}
		return true
	}

	var confirmed *csi.ValidateVolumeCapabilitiesResponse_Confirmed
	if isValidVolumeCapabilities(volCaps) {
		confirmed = &csi.ValidateVolumeCapabilitiesResponse_Confirmed{VolumeCapabilities: volCaps}
	}
	return &csi.ValidateVolumeCapabilitiesResponse{Confirmed: confirmed}, nil
}

func (c *controllerService) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	klog.V(4).Infof("ListVolumes: %+v", *req)
	return nil, status.Error(codes.Unimplemented, "ListVolumes is not supported")
}

func (c *controllerService) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	klog.V(4).Infof("GetCapacity: %+v", *req)
	return nil, status.Error(codes.Unimplemented, "GetCapacity is not supported")
}

func (c *controllerService) ControllerGetCapabilities(cxt context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	klog.V(4).Infof("ControllerGetCapabilities: %+v", *req)
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
					},
				},
			},
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
					},
				},
			},
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
					},
				},
			},
		},
	}, nil
}

func (c *controllerService) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	klog.V(4).Infof("CreateSnapshot: %+v", *req)
	return nil, status.Error(codes.Unimplemented, "CreateSnapshot is not supported")
}

func (c *controllerService) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	klog.V(4).Infof("DeleteSnapshot: %+v", *req)
	return nil, status.Error(codes.Unimplemented, "DeleteSnapshot is not supported")
}

func (c *controllerService) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	klog.V(4).Infof("ListSnapshots: %+v", *req)
	return nil, status.Error(codes.Unimplemented, "ListSnapshots is not supported")
}

func (c *controllerService) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	klog.V(4).Infof("ControllerExpandVolume: %+v", *req)
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID must be provided")
	}
	volume, err := c.client.Volumes.Get(ctx, volumeID)
	if err != nil {
		if err == ah.ErrResourceNotFound {
			return nil, status.Errorf(codes.NotFound, "Volume %s not found", volumeID)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	capRange := req.GetCapacityRange()
	if capRange == nil {
		return nil, status.Error(codes.InvalidArgument, "Capacity range must be provided")
	}

	newSize, err := volumeSize(capRange)
	if err != nil {
		return nil, status.Error(codes.OutOfRange, "Volume size exceeds the limit")
	}

	if newSize <= int64(volume.Size) {
		return &csi.ControllerExpandVolumeResponse{CapacityBytes: int64(volume.Size * gb), NodeExpansionRequired: true}, nil
	}

	action, err := c.client.Volumes.Resize(ctx, volumeID, int(newSize))
	if err != nil {
		if strings.Contains(err.Error(), "performing another action: Resize") {
			return nil, status.Errorf(codes.Aborted, err.Error())
		}
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if err := c.waitForVolumeActionState(ctx, volumeID, action.ID, "success"); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &csi.ControllerExpandVolumeResponse{
		CapacityBytes:         newSize * gb,
		NodeExpansionRequired: true,
	}, nil
}

func (c *controllerService) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	klog.V(4).Infof("ControllerGetVolume: %+v", *req)
	return nil, status.Errorf(codes.Unimplemented, "ControllerGetVolume is not supported")
}

func (c *controllerService) waitForVolumeActionState(ctx context.Context, volumeID, actionID, state string) error {
	stateFunc := func(ctx context.Context) (state string, err error) {
		v, err := c.client.Volumes.ActionInfo(ctx, volumeID, actionID)
		if err != nil {
			return "", err
		}
		if v.State == "failed" {
			return "", fmt.Errorf("volume %s in the error state", v.ID)
		}
		return v.State, nil
	}
	if err := waitForState(ctx, stateFunc, state); err != nil {
		return err
	}
	return nil
}

func (c *controllerService) waitForCloudServerActionState(ctx context.Context, cloudServerID, actionID, state string) error {
	stateFunc := func(ctx context.Context) (state string, err error) {
		v, err := c.client.Instances.ActionInfo(ctx, cloudServerID, actionID)
		if err != nil {
			return "", err
		}
		if v.State == "failed" {
			return "", fmt.Errorf("volume %s in the error state", v.ID)
		}
		return v.State, nil
	}
	if err := waitForState(ctx, stateFunc, state); err != nil {
		return err
	}
	return nil
}

func (c *controllerService) waitForVolumeState(ctx context.Context, volumeID, state string) error {
	stateFunc := func(ctx context.Context) (state string, err error) {
		v, err := c.client.Volumes.Get(ctx, volumeID)
		if err != nil {
			if err == ah.ErrResourceNotFound {
				return "deleted", nil
			}
			return "", err
		}
		if v.State == "failed" {
			return "", fmt.Errorf("volume %s in the error state", v.ID)
		}
		return v.State, nil
	}
	if err := waitForState(ctx, stateFunc, state); err != nil {
		return err
	}
	return nil
}
