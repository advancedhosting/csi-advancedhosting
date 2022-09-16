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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"reflect"
	"testing"

	"github.com/advancedhosting/csi-advancedhosting/driver/mocks"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/mock/gomock"
)

func TestControllerCreateVolume(t *testing.T) {

	capRange := &csi.CapacityRange{RequiredBytes: 10 * GiB}
	volCap := []*csi.VolumeCapability{
		{
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}
	invalidVolCap := []*csi.VolumeCapability{
		{
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
			},
		},
	}
	params := map[string]string{VolumeProductSlugKey: "test-product-slug"}

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "ControllerCreateVolume Success",
			testFunc: func(t *testing.T) {
				req := &csi.CreateVolumeRequest{
					Name:               "test-name",
					CapacityRange:      capRange,
					VolumeCapabilities: volCap,
					Parameters:         params,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				filter := &ah.EqFilter{
					Keys:  []string{"name"},
					Value: "test-name",
				}
				options := &ah.ListOptions{
					Filters: []ah.FilterInterface{filter},
				}
				mockedVolumesAPI.EXPECT().List(gomock.Eq(ctx), gomock.Eq(options)).Return([]ah.Volume{}, nil, nil)

				createRequest := &ah.VolumeCreateRequest{
					Name:        "test-name",
					Size:        10,
					ProductSlug: "test-product-slug",
				}
				returnValue := &ah.Volume{
					ID:    "volumeID",
					Size:  10,
					State: "creating",
				}
				returnValue.VolumePool = (*struct {
					Name             string   `json:"name,omitempty"` //nolint
					DatacenterIDs    []string `json:"datacenter_ids,omitempty"`
					ReplicationLevel int      `json:"replication_level,omitempty"` //nolint
				})(&struct {
					Name             string //nolint
					DatacenterIDs    []string
					ReplicationLevel int //nolint
				}{DatacenterIDs: []string{"test-datacenter-id"}})
				mockedVolumesAPI.EXPECT().Create(gomock.Eq(ctx), gomock.Eq(createRequest)).Return(returnValue, nil)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(returnValue.ID)).Return(&ah.Volume{State: "creating"}, nil)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(returnValue.ID)).Return(&ah.Volume{State: "ready"}, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				response, err := controllerSvc.CreateVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				volume := response.GetVolume()
				if volume.GetVolumeId() != "volumeID" {
					t.Errorf("Unexpected volume id: %v", volume.GetVolumeId())
				}
				if volume.GetCapacityBytes() != 10*GiB {
					t.Errorf("Unexpected capacity: %v", volume.GetCapacityBytes())
				}

				datacenterID := volume.GetAccessibleTopology()[0].GetSegments()[TopologySegmentDatacenter]
				if datacenterID != "test-datacenter-id" {
					t.Errorf("Unexpected datacenter: %v", datacenterID)
				}
			},
		},
		{
			name: "ControllerCreateVolume Success with cluster-id",
			testFunc: func(t *testing.T) {
				req := &csi.CreateVolumeRequest{
					Name:               "test-name",
					CapacityRange:      capRange,
					VolumeCapabilities: volCap,
					Parameters:         params,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				filter := &ah.EqFilter{
					Keys:  []string{"name"},
					Value: "kube-test-cluster-id-test-name",
				}
				options := &ah.ListOptions{
					Filters: []ah.FilterInterface{filter},
				}
				mockedVolumesAPI.EXPECT().List(gomock.Eq(ctx), gomock.Eq(options)).Return([]ah.Volume{}, nil, nil)

				createRequest := &ah.VolumeCreateRequest{
					Name:        "kube-test-cluster-id-test-name",
					Size:        10,
					ProductSlug: "test-product-slug",
				}
				returnValue := &ah.Volume{
					ID:    "volumeID",
					Size:  10,
					State: "creating",
				}
				returnValue.VolumePool = (*struct {
					Name             string   `json:"name,omitempty"` //nolint
					DatacenterIDs    []string `json:"datacenter_ids,omitempty"`
					ReplicationLevel int      `json:"replication_level,omitempty"` //nolint
				})(&struct {
					Name             string //nolint
					DatacenterIDs    []string
					ReplicationLevel int //nolint
				}{DatacenterIDs: []string{"test-datacenter-id"}})
				mockedVolumesAPI.EXPECT().Create(gomock.Eq(ctx), gomock.Eq(createRequest)).Return(returnValue, nil)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(returnValue.ID)).Return(&ah.Volume{State: "creating"}, nil)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(returnValue.ID)).Return(&ah.Volume{State: "ready"}, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "test-cluster-id")

				response, err := controllerSvc.CreateVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				volume := response.GetVolume()
				if volume.GetVolumeId() != "volumeID" {
					t.Errorf("Unexpected volume id: %v", volume.GetVolumeId())
				}
				if volume.GetCapacityBytes() != 10*GiB {
					t.Errorf("Unexpected capacity: %v", volume.GetCapacityBytes())
				}

				datacenterID := volume.GetAccessibleTopology()[0].GetSegments()[TopologySegmentDatacenter]
				if datacenterID != "test-datacenter-id" {
					t.Errorf("Unexpected datacenter: %v", datacenterID)
				}
			},
		},
		{
			name: "ControllerCreateVolume Empty name",
			testFunc: func(t *testing.T) {
				req := &csi.CreateVolumeRequest{
					Name:               "",
					CapacityRange:      capRange,
					VolumeCapabilities: volCap,
					Parameters:         params,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.CreateVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument || respErr.Message() != "Volume name not provided" {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}

			},
		},
		{
			name: "ControllerCreateVolume Without product slug",
			testFunc: func(t *testing.T) {
				req := &csi.CreateVolumeRequest{
					Name:               "test",
					CapacityRange:      capRange,
					VolumeCapabilities: volCap,
					Parameters:         nil,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.CreateVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument || respErr.Message() != "product-slug param must be provided" {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}

			},
		},
		{
			name: "ControllerCreateVolume Empty capabilities",
			testFunc: func(t *testing.T) {
				req := &csi.CreateVolumeRequest{
					Name:               "test",
					CapacityRange:      capRange,
					VolumeCapabilities: []*csi.VolumeCapability{},
					Parameters:         params,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.CreateVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}

			},
		},
		{
			name: "ControllerCreateVolume Invalid capabilities",
			testFunc: func(t *testing.T) {
				req := &csi.CreateVolumeRequest{
					Name:               "test",
					CapacityRange:      capRange,
					VolumeCapabilities: invalidVolCap,
					Parameters:         params,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.CreateVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}

			},
		},
		{
			name: "ControllerCreateVolume Invalid size",
			testFunc: func(t *testing.T) {
				req := &csi.CreateVolumeRequest{
					Name: "test",
					CapacityRange: &csi.CapacityRange{
						RequiredBytes: 10*GiB + 1,
						LimitBytes:    10 * GiB,
					},
					VolumeCapabilities: volCap,
					Parameters:         params,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.CreateVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.OutOfRange {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}

			},
		},
		{
			name: "ControllerCreateVolume Multiple volumes already exist",
			testFunc: func(t *testing.T) {
				req := &csi.CreateVolumeRequest{
					Name:               "test-name",
					CapacityRange:      capRange,
					VolumeCapabilities: volCap,
					Parameters:         params,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				filter := &ah.EqFilter{
					Keys:  []string{"name"},
					Value: "test-name",
				}
				options := &ah.ListOptions{
					Filters: []ah.FilterInterface{filter},
				}

				listResponse := []ah.Volume{
					{
						ID: "volumeID",
					},
					{
						ID: "volumeID2",
					},
				}

				mockedVolumesAPI.EXPECT().List(gomock.Eq(ctx), gomock.Eq(options)).Return(listResponse, nil, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.CreateVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.AlreadyExists {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}

			},
		},
		{
			name: "ControllerCreateVolume Volume already exists with diff size",
			testFunc: func(t *testing.T) {
				req := &csi.CreateVolumeRequest{
					Name:               "test-name",
					CapacityRange:      capRange,
					VolumeCapabilities: volCap,
					Parameters:         params,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				filter := &ah.EqFilter{
					Keys:  []string{"name"},
					Value: "test-name",
				}
				options := &ah.ListOptions{
					Filters: []ah.FilterInterface{filter},
				}

				listResponse := []ah.Volume{
					{
						ID:   "volumeID",
						Size: 12,
					},
				}

				mockedVolumesAPI.EXPECT().List(gomock.Eq(ctx), gomock.Eq(options)).Return(listResponse, nil, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.CreateVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.AlreadyExists {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}

			},
		},

		{
			name: "ControllerCreateVolume Volume already exists",
			testFunc: func(t *testing.T) {
				req := &csi.CreateVolumeRequest{
					Name:               "test-name",
					CapacityRange:      capRange,
					VolumeCapabilities: volCap,
					Parameters:         params,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				filter := &ah.EqFilter{
					Keys:  []string{"name"},
					Value: "test-name",
				}
				options := &ah.ListOptions{
					Filters: []ah.FilterInterface{filter},
				}

				returnValue := &ah.Volume{
					ID:   "volumeID",
					Size: 10,
				}
				returnValue.VolumePool = (*struct {
					Name             string   `json:"name,omitempty"` //nolint
					DatacenterIDs    []string `json:"datacenter_ids,omitempty"`
					ReplicationLevel int      `json:"replication_level,omitempty"` //nolint
				})(&struct {
					Name             string //nolint
					DatacenterIDs    []string
					ReplicationLevel int //nolint
				}{DatacenterIDs: []string{"test-datacenter-id"}})

				listResponse := []ah.Volume{*returnValue}

				mockedVolumesAPI.EXPECT().List(gomock.Eq(ctx), gomock.Eq(options)).Return(listResponse, nil, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				response, err := controllerSvc.CreateVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				volume := response.GetVolume()
				if volume.GetVolumeId() != "volumeID" {
					t.Errorf("Unexpected volume id: %v", volume.GetVolumeId())
				}
				if volume.GetCapacityBytes() != 10*GiB {
					t.Errorf("Unexpected capacity: %v", volume.GetCapacityBytes())
				}

				datacenterID := volume.GetAccessibleTopology()[0].GetSegments()[TopologySegmentDatacenter]
				if datacenterID != "test-datacenter-id" {
					t.Errorf("Unexpected datacenter: %v", datacenterID)
				}

			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestControllerDeleteVolume(t *testing.T) {

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "ControllerDeleteVolume Success",
			testFunc: func(t *testing.T) {
				req := &csi.DeleteVolumeRequest{
					VolumeId: "test-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedVolumesAPI.EXPECT().Delete(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				resp, err := controllerSvc.DeleteVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.DeleteVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}

			},
		},
		{
			name: "ControllerDeleteVolume Empty volume id",
			testFunc: func(t *testing.T) {
				req := &csi.DeleteVolumeRequest{
					VolumeId: "",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.DeleteVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
		{
			name: "ControllerDeleteVolume Volume is already removed",
			testFunc: func(t *testing.T) {
				req := &csi.DeleteVolumeRequest{
					VolumeId: "test-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedVolumesAPI.EXPECT().Delete(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(ah.ErrResourceNotFound)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				resp, err := controllerSvc.DeleteVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.DeleteVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "ControllerDeleteVolume Handle concurrency",
			testFunc: func(t *testing.T) {
				req := &csi.DeleteVolumeRequest{
					VolumeId: "test-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				deleteErr := fmt.Errorf("\"base\": [\"Volume is currently busy performing another action: Destroy\"]")
				mockedVolumesAPI.EXPECT().Delete(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(deleteErr)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.DeleteVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.Aborted {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestControllerPublishVolume(t *testing.T) {
	volCap := &csi.VolumeCapability{
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		},
	}

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "ControllerPublishVolume Success",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerPublishVolumeRequest{
					VolumeId:         "test-id",
					NodeId:           "test-node-id",
					VolumeCapability: volCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				volume := &ah.Volume{
					ID: "test-id",
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				action := &ah.Action{ID: "test-action-id"}
				mockedInstancesAPI.EXPECT().AttachVolume(gomock.Eq(ctx), gomock.Eq(req.NodeId), gomock.Eq(req.VolumeId)).Return(action, nil)
				mockedInstancesAPI.EXPECT().ActionInfo(gomock.Eq(ctx), gomock.Eq(req.NodeId), gomock.Eq(action.ID)).Return(&ah.InstanceAction{Action: &ah.Action{State: "running"}}, nil)
				mockedInstancesAPI.EXPECT().ActionInfo(gomock.Eq(ctx), gomock.Eq(req.NodeId), gomock.Eq(action.ID)).Return(&ah.InstanceAction{Action: &ah.Action{State: "success"}}, nil)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				resp, err := controllerSvc.ControllerPublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.ControllerPublishVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "ControllerPublishVolume Empty volume id",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerPublishVolumeRequest{
					VolumeId:         "",
					NodeId:           "test-node-id",
					VolumeCapability: volCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerPublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}

			},
		},
		{
			name: "ControllerPublishVolume Empty node id",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerPublishVolumeRequest{
					VolumeId:         "test-id",
					NodeId:           "",
					VolumeCapability: volCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerPublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}

			},
		},
		{
			name: "ControllerPublishVolume Volume id doesn't exist",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerPublishVolumeRequest{
					VolumeId:         "test-id",
					NodeId:           "test-node-id",
					VolumeCapability: volCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(nil, ah.ErrResourceNotFound)

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerPublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.NotFound {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}

			},
		},
		{
			name: "ControllerPublishVolume Node id doesn't exist",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerPublishVolumeRequest{
					VolumeId:         "test-id",
					NodeId:           "test-node-id",
					VolumeCapability: volCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}
				volume := &ah.Volume{
					ID: "test-id",
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedInstancesAPI.EXPECT().AttachVolume(gomock.Eq(ctx), gomock.Eq(req.NodeId), gomock.Eq(req.VolumeId)).Return(nil, ah.ErrResourceNotFound)

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerPublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.NotFound {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}

			},
		},
		{
			name: "ControllerPublishVolume Volume is attached to the another volume",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerPublishVolumeRequest{
					VolumeId:         "test-id",
					NodeId:           "test-node-id",
					VolumeCapability: volCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				volume := &ah.Volume{
					ID: "test-id",
				}
				volume.Instance = (*struct {
					ID   string `json:"id,omitempty"`
					Name string `json:"name,omitempty"`
				})(&struct {
					ID   string
					Name string
				}{ID: "random-id"})
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerPublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.FailedPrecondition {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
		{
			name: "ControllerPublishVolume Volume is already attached to node",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerPublishVolumeRequest{
					VolumeId:         "test-id",
					NodeId:           "test-node-id",
					VolumeCapability: volCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				volume := &ah.Volume{
					ID: "test-id",
				}
				volume.Instance = (*struct {
					ID   string `json:"id,omitempty"`
					Name string `json:"name,omitempty"` //nolint:unused
				})(&struct {
					ID   string
					Name string //nolint:unused
				}{ID: "test-node-id"})
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				resp, err := controllerSvc.ControllerPublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.ControllerPublishVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "ControllerPublishVolume Maximum volumes",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerPublishVolumeRequest{
					VolumeId:         "test-id",
					NodeId:           "test-node-id",
					VolumeCapability: volCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				volume := &ah.Volume{
					ID: "test-id",
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedInstancesAPI.EXPECT().AttachVolume(gomock.Eq(ctx), gomock.Eq(req.NodeId), gomock.Eq(req.VolumeId)).Return(nil, fmt.Errorf("Reached maximum volumes number"))

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerPublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.ResourceExhausted {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
		{
			name: "ControllerPublishVolume Handle concurrency",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerPublishVolumeRequest{
					VolumeId:         "test-id",
					NodeId:           "test-node-id",
					VolumeCapability: volCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				volume := &ah.Volume{
					ID: "test-id",
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				attachErr := fmt.Errorf("performing another action: Attach volume")
				mockedInstancesAPI.EXPECT().AttachVolume(gomock.Eq(ctx), gomock.Eq(req.NodeId), gomock.Eq(req.VolumeId)).Return(nil, attachErr)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerPublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.Aborted {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestControllerUnpublishVolume(t *testing.T) {
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "ControllerUnpublishVolume Success",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerUnpublishVolumeRequest{
					VolumeId: "test-id",
					NodeId:   "test-node-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				volume := &ah.Volume{
					ID: "test-id",
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				action := &ah.Action{ID: "test-action-id"}
				mockedInstancesAPI.EXPECT().DetachVolume(gomock.Eq(ctx), gomock.Eq(req.NodeId), gomock.Eq(req.VolumeId)).Return(action, nil)
				mockedInstancesAPI.EXPECT().ActionInfo(gomock.Eq(ctx), gomock.Eq(req.NodeId), gomock.Eq(action.ID)).Return(&ah.InstanceAction{Action: &ah.Action{State: "running"}}, nil)
				mockedInstancesAPI.EXPECT().ActionInfo(gomock.Eq(ctx), gomock.Eq(req.NodeId), gomock.Eq(action.ID)).Return(&ah.InstanceAction{Action: &ah.Action{State: "success"}}, nil)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				resp, err := controllerSvc.ControllerUnpublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.ControllerUnpublishVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "ControllerUnpublishVolume Empty volume id",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerUnpublishVolumeRequest{
					VolumeId: "",
					NodeId:   "test-node-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerUnpublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}

			},
		},
		{
			name: "ControllerUnpublishVolume Empty node id",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerUnpublishVolumeRequest{
					VolumeId: "test-id",
					NodeId:   "",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)
				volume := &ah.Volume{
					ID: "test-id",
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerUnpublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
		{
			name: "ControllerUnpublishVolume Volume doesn't exist",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerUnpublishVolumeRequest{
					VolumeId: "test-id",
					NodeId:   "test-node-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(nil, ah.ErrResourceNotFound)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				resp, err := controllerSvc.ControllerUnpublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.ControllerUnpublishVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "ControllerUnpublishVolume Node doesn't exist",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerUnpublishVolumeRequest{
					VolumeId: "test-id",
					NodeId:   "test-node-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				volume := &ah.Volume{
					ID: "test-id",
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedInstancesAPI.EXPECT().DetachVolume(gomock.Eq(ctx), gomock.Eq(req.NodeId), gomock.Eq(req.VolumeId)).Return(nil, ah.ErrResourceNotFound)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				resp, err := controllerSvc.ControllerUnpublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.ControllerUnpublishVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "ControllerUnpublishVolume Already detached",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerUnpublishVolumeRequest{
					VolumeId: "test-id",
					NodeId:   "test-node-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				volume := &ah.Volume{
					ID: "test-id",
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedInstancesAPI.EXPECT().DetachVolume(gomock.Eq(ctx), gomock.Eq(req.NodeId), gomock.Eq(req.VolumeId)).Return(nil, fmt.Errorf("{\"volume\":[\"not attached to the instance\",\"not attached to the current instance\"]}"))

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				resp, err := controllerSvc.ControllerUnpublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.ControllerUnpublishVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "ControllerUnpublishVolume Handle concurrency",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerUnpublishVolumeRequest{
					VolumeId: "test-id",
					NodeId:   "test-node-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedInstancesAPI := mocks.NewMockInstancesAPI(ctrl)

				volume := &ah.Volume{
					ID: "test-id",
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				detachErr := fmt.Errorf("performing another action: Detach volume")
				mockedInstancesAPI.EXPECT().DetachVolume(gomock.Eq(ctx), gomock.Eq(req.NodeId), gomock.Eq(req.VolumeId)).Return(nil, detachErr)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI, Instances: mockedInstancesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerUnpublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.Aborted {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestControllerValidateVolumeCapabilities(t *testing.T) {
	volCaps := []*csi.VolumeCapability{
		{
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}
	invalidVolCaps := []*csi.VolumeCapability{
		{
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
			},
		},
		{
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "ValidateVolumeCapabilities Success",
			testFunc: func(t *testing.T) {
				req := &csi.ValidateVolumeCapabilitiesRequest{
					VolumeId:           "test-id",
					VolumeCapabilities: volCaps,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				volume := &ah.Volume{
					ID: "test-id",
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				resp, err := controllerSvc.ValidateVolumeCapabilities(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				confirmed := &csi.ValidateVolumeCapabilitiesResponse_Confirmed{VolumeCapabilities: volCaps}
				expectedResult := &csi.ValidateVolumeCapabilitiesResponse{Confirmed: confirmed}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "ValidateVolumeCapabilities Empty volume id",
			testFunc: func(t *testing.T) {
				req := &csi.ValidateVolumeCapabilitiesRequest{
					VolumeId:           "",
					VolumeCapabilities: volCaps,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ValidateVolumeCapabilities(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
		{
			name: "ValidateVolumeCapabilities Volume not found",
			testFunc: func(t *testing.T) {
				req := &csi.ValidateVolumeCapabilitiesRequest{
					VolumeId:           "test-id",
					VolumeCapabilities: volCaps,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(nil, ah.ErrResourceNotFound)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ValidateVolumeCapabilities(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.NotFound {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
		{
			name: "ValidateVolumeCapabilities Empty capabilities",
			testFunc: func(t *testing.T) {
				req := &csi.ValidateVolumeCapabilitiesRequest{
					VolumeId: "test-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ValidateVolumeCapabilities(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
		{
			name: "ValidateVolumeCapabilities Invalid capabilities",
			testFunc: func(t *testing.T) {
				req := &csi.ValidateVolumeCapabilitiesRequest{
					VolumeId:           "test-id",
					VolumeCapabilities: invalidVolCaps,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				resp, err := controllerSvc.ValidateVolumeCapabilities(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				expectedResult := &csi.ValidateVolumeCapabilitiesResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestControllerExpandVolume(t *testing.T) {
	capRange := &csi.CapacityRange{RequiredBytes: 10 * GiB}

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "ControllerExpandVolume Success",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerExpandVolumeRequest{
					VolumeId:      "test-id",
					CapacityRange: capRange,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				volume := &ah.Volume{
					ID:   "test-id",
					Size: 5,
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				action := &ah.Action{
					ID: "action-id",
				}
				mockedVolumesAPI.EXPECT().Resize(gomock.Eq(ctx), gomock.Eq(req.VolumeId), gomock.Eq(10)).Return(action, nil)
				mockedVolumesAPI.EXPECT().ActionInfo(
					gomock.Eq(ctx), gomock.Eq(req.VolumeId), gomock.Eq(action.ID)).Return(&ah.VolumeAction{Action: &ah.Action{State: "running_allocation"}}, nil)
				mockedVolumesAPI.EXPECT().ActionInfo(
					gomock.Eq(ctx), gomock.Eq(req.VolumeId), gomock.Eq(action.ID)).Return(&ah.VolumeAction{Action: &ah.Action{State: "success"}}, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				resp, err := controllerSvc.ControllerExpandVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				expectedResult := &csi.ControllerExpandVolumeResponse{
					CapacityBytes:         10 * GiB,
					NodeExpansionRequired: true,
				}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "ControllerExpandVolume Empty volume id",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerExpandVolumeRequest{
					CapacityRange: capRange,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerExpandVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
		{
			name: "ControllerExpandVolume Volume id not found",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerExpandVolumeRequest{
					VolumeId:      "test-id",
					CapacityRange: capRange,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(nil, ah.ErrResourceNotFound)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerExpandVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.NotFound {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
		{
			name: "ControllerExpandVolume Empty CapacityRange",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerExpandVolumeRequest{
					VolumeId: "test-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerExpandVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
		{
			name: "ControllerExpandVolume Invalid CapacityRange",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerExpandVolumeRequest{
					VolumeId:      "test-id",
					CapacityRange: &csi.CapacityRange{RequiredBytes: 1000, LimitBytes: 5000},
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerExpandVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.OutOfRange {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
		{
			name: "ControllerExpandVolume New size less than existed volume size",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerExpandVolumeRequest{
					VolumeId:      "test-id",
					CapacityRange: capRange,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				volume := &ah.Volume{
					ID:   "test-id",
					Size: 20,
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				resp, err := controllerSvc.ControllerExpandVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				expectedResult := &csi.ControllerExpandVolumeResponse{
					CapacityBytes:         int64(volume.Size * GiB),
					NodeExpansionRequired: true,
				}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "ControllerExpandVolume Failed while resizing",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerExpandVolumeRequest{
					VolumeId:      "test-id",
					CapacityRange: capRange,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				volume := &ah.Volume{
					ID:   "test-id",
					Size: 5,
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				action := &ah.Action{
					ID: "action-id",
				}
				mockedVolumesAPI.EXPECT().Resize(gomock.Eq(ctx), gomock.Eq(req.VolumeId), gomock.Eq(10)).Return(action, nil)
				mockedVolumesAPI.EXPECT().ActionInfo(
					gomock.Eq(ctx), gomock.Eq(req.VolumeId), gomock.Eq(action.ID)).Return(&ah.VolumeAction{Action: &ah.Action{State: "running_allocation"}}, nil)
				mockedVolumesAPI.EXPECT().ActionInfo(
					gomock.Eq(ctx), gomock.Eq(req.VolumeId), gomock.Eq(action.ID)).Return(&ah.VolumeAction{Action: &ah.Action{State: "failed"}}, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerExpandVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.Internal {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
		{
			name: "ControllerExpandVolume Handle concurrency",
			testFunc: func(t *testing.T) {
				req := &csi.ControllerExpandVolumeRequest{
					VolumeId:      "test-id",
					CapacityRange: capRange,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				volume := &ah.Volume{
					ID:   "test-id",
					Size: 5,
				}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)

				resizeErr := fmt.Errorf("performing another action: Resize")
				mockedVolumesAPI.EXPECT().Resize(gomock.Eq(ctx), gomock.Eq(req.VolumeId), gomock.Eq(10)).Return(nil, resizeErr)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				controllerSvc := NewControllerService(mockedClient, "")

				r, err := controllerSvc.ControllerExpandVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.Aborted {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}
