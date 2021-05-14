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
	"github.com/advancedhosting/advancedhosting-api-go/ah"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/advancedhosting/csi-advancedhosting/driver/mocks"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/mock/gomock"
)

func TestNodeStageVolume(t *testing.T) {
	fsType := "ext3"
	volCap := &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{
				FsType: fsType,
			},
		},
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		},
	}
	invalidVolCap := &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{
				FsType: fsType,
			},
		},
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
		},
	}

	sourcePath := "/source/path"

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "NodeStageVolume Success",
			testFunc: func(t *testing.T) {
				req := &csi.NodeStageVolumeRequest{
					VolumeId:          "test-id",
					VolumeCapability:  volCap,
					StagingTargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				volume := &ah.Volume{Number: "test-volume-number"}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedMounter.EXPECT().DevicePath(gomock.Eq(volume.Number)).Return(sourcePath, nil)
				mockedMounter.EXPECT().IsDeviceMountedToTarget(gomock.Eq(sourcePath), gomock.Eq(req.StagingTargetPath)).Return(false, nil)
				mockedMounter.EXPECT().IsPathExists(gomock.Eq(req.StagingTargetPath)).Return(false, nil)
				mockedMounter.EXPECT().CreateDir(gomock.Eq(req.StagingTargetPath)).Return(nil)
				mockedMounter.EXPECT().FormatAndMount(gomock.Eq(sourcePath), gomock.Eq(req.StagingTargetPath), gomock.Eq(fsType), gomock.Eq([]string(nil))).Return(nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeStageVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeStageVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodeStageVolume Already mounted",
			testFunc: func(t *testing.T) {
				req := &csi.NodeStageVolumeRequest{
					VolumeId:          "test-id",
					VolumeCapability:  volCap,
					StagingTargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				volume := &ah.Volume{Number: "test-volume-number"}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedMounter.EXPECT().DevicePath(gomock.Eq(volume.Number)).Return(sourcePath, nil)
				mockedMounter.EXPECT().IsDeviceMountedToTarget(gomock.Eq(sourcePath), gomock.Eq(req.StagingTargetPath)).Return(true, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeStageVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeStageVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodeStageVolume With options",
			testFunc: func(t *testing.T) {
				mountFlags := []string{"flag1", "flag2"}
				volBlockCap := &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{
							FsType:     fsType,
							MountFlags: mountFlags,
						},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				}

				req := &csi.NodeStageVolumeRequest{
					VolumeId:          "test-id",
					VolumeCapability:  volBlockCap,
					StagingTargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				volume := &ah.Volume{Number: "test-volume-number"}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedMounter.EXPECT().DevicePath(gomock.Eq(volume.Number)).Return(sourcePath, nil)
				mockedMounter.EXPECT().IsDeviceMountedToTarget(gomock.Eq(sourcePath), gomock.Eq(req.StagingTargetPath)).Return(false, nil)
				mockedMounter.EXPECT().IsPathExists(gomock.Eq(req.StagingTargetPath)).Return(false, nil)
				mockedMounter.EXPECT().CreateDir(gomock.Eq(req.StagingTargetPath)).Return(nil)
				mockedMounter.EXPECT().FormatAndMount(gomock.Eq(sourcePath), gomock.Eq(req.StagingTargetPath), gomock.Eq(fsType), gomock.Eq(mountFlags)).Return(nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeStageVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeStageVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodeStageVolume Empty volume id",
			testFunc: func(t *testing.T) {
				req := &csi.NodeStageVolumeRequest{
					VolumeId:          "",
					VolumeCapability:  volCap,
					StagingTargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeStageVolume(ctx, req)
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
			name: "NodeStageVolume Volume not found",
			testFunc: func(t *testing.T) {
				req := &csi.NodeStageVolumeRequest{
					VolumeId:          "test-id",
					VolumeCapability:  volCap,
					StagingTargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(nil, ah.ErrResourceNotFound)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeStageVolume(ctx, req)
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
			name: "NodeStageVolume Empty target path",
			testFunc: func(t *testing.T) {
				req := &csi.NodeStageVolumeRequest{
					VolumeId:          "test-id",
					VolumeCapability:  volCap,
					StagingTargetPath: "",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeStageVolume(ctx, req)
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
			name: "NodeStageVolume Empty capability",
			testFunc: func(t *testing.T) {
				req := &csi.NodeStageVolumeRequest{
					VolumeId:          "test-id",
					StagingTargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeStageVolume(ctx, req)
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
			name: "NodeStageVolume Invalid capability",
			testFunc: func(t *testing.T) {
				req := &csi.NodeStageVolumeRequest{
					VolumeId:          "test-id",
					StagingTargetPath: "/target/path",
					VolumeCapability:  invalidVolCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeStageVolume(ctx, req)
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
			name: "NodeStageVolume Block capability",
			testFunc: func(t *testing.T) {

				volBlockCap := &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Block{
						Block: &csi.VolumeCapability_BlockVolume{},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				}

				req := &csi.NodeStageVolumeRequest{
					VolumeId:          "test-id",
					StagingTargetPath: "/target/path",
					VolumeCapability:  volBlockCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeStageVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeStageVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodeStageVolume Without access type",
			testFunc: func(t *testing.T) {

				volBlockCap := &csi.VolumeCapability{
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				}

				req := &csi.NodeStageVolumeRequest{
					VolumeId:          "test-id",
					StagingTargetPath: "/target/path",
					VolumeCapability:  volBlockCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeStageVolume(ctx, req)
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
			name: "NodeStageVolume Without FS",
			testFunc: func(t *testing.T) {
				volBlockCap := &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				}
				req := &csi.NodeStageVolumeRequest{
					VolumeId:          "test-id",
					VolumeCapability:  volBlockCap,
					StagingTargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				volume := &ah.Volume{Number: "test-volume-number"}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedMounter.EXPECT().DevicePath(gomock.Eq(volume.Number)).Return(sourcePath, nil)
				mockedMounter.EXPECT().IsDeviceMountedToTarget(gomock.Eq(sourcePath), gomock.Eq(req.StagingTargetPath)).Return(false, nil)
				mockedMounter.EXPECT().IsPathExists(gomock.Eq(req.StagingTargetPath)).Return(false, nil)
				mockedMounter.EXPECT().CreateDir(gomock.Eq(req.StagingTargetPath)).Return(nil)
				mockedMounter.EXPECT().FormatAndMount(gomock.Eq(sourcePath), gomock.Eq(req.StagingTargetPath), gomock.Eq(DefaultFSType), gomock.Eq([]string(nil))).Return(nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeStageVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeStageVolumeResponse{}
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

func TestNodeUnstageVolume(t *testing.T) {
	sourcePath := "/source/path"
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "NodeUnstageVolume Success",
			testFunc: func(t *testing.T) {
				req := &csi.NodeUnstageVolumeRequest{
					VolumeId:          "test-id",
					StagingTargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				volume := &ah.Volume{Number: "test-volume-number"}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedMounter.EXPECT().DevicePath(gomock.Eq(volume.Number)).Return(sourcePath, nil)
				mockedMounter.EXPECT().IsDeviceMountedToTarget(gomock.Eq(sourcePath), gomock.Eq(req.StagingTargetPath)).Return(true, nil)
				mockedMounter.EXPECT().Unmount(gomock.Eq(req.StagingTargetPath)).Return(nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeUnstageVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeUnstageVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodeUnstageVolume Not mounted",
			testFunc: func(t *testing.T) {
				req := &csi.NodeUnstageVolumeRequest{
					VolumeId:          "test-id",
					StagingTargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				volume := &ah.Volume{Number: "test-volume-number"}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedMounter.EXPECT().DevicePath(gomock.Eq(volume.Number)).Return(sourcePath, nil)
				mockedMounter.EXPECT().IsDeviceMountedToTarget(gomock.Eq(sourcePath), gomock.Eq(req.StagingTargetPath)).Return(false, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeUnstageVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeUnstageVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodeUnstageVolume Empty volume id",
			testFunc: func(t *testing.T) {
				req := &csi.NodeUnstageVolumeRequest{
					VolumeId:          "",
					StagingTargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}
				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeUnstageVolume(ctx, req)
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
			name: "NodeUnstageVolume Empty staging target path",
			testFunc: func(t *testing.T) {
				req := &csi.NodeUnstageVolumeRequest{
					VolumeId: "test-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeUnstageVolume(ctx, req)
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
			name: "NodeUnstageVolume Volume not found",
			testFunc: func(t *testing.T) {
				req := &csi.NodeUnstageVolumeRequest{
					VolumeId:          "test-id",
					StagingTargetPath: "/test/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(nil, ah.ErrResourceNotFound)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeUnstageVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.NotFound {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestNodePublishVolume(t *testing.T) {
	fsType := "ext3"
	volMountCap := &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{
				FsType: fsType,
			},
		},
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		},
	}
	volBlockCap := &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Block{
			Block: &csi.VolumeCapability_BlockVolume{},
		},
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		},
	}
	sourcePath := "/source/path"

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "NodePublishVolume Success mount",
			testFunc: func(t *testing.T) {
				req := &csi.NodePublishVolumeRequest{
					VolumeId:          "test-id",
					VolumeCapability:  volMountCap,
					StagingTargetPath: "/staging/target/path",
					TargetPath:        "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				volume := &ah.Volume{Number: "test-volume-number"}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedMounter.EXPECT().CreateDir(gomock.Eq(req.TargetPath)).Return(nil)
				mockedMounter.EXPECT().Mount(gomock.Eq(req.StagingTargetPath), gomock.Eq(req.TargetPath), gomock.Eq(fsType), gomock.Eq([]string{"bind"})).Return(nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodePublishVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodePublishVolume Success read only mount",
			testFunc: func(t *testing.T) {
				req := &csi.NodePublishVolumeRequest{
					VolumeId:          "test-id",
					VolumeCapability:  volMountCap,
					StagingTargetPath: "/staging/target/path",
					TargetPath:        "/target/path",
					Readonly:          true,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				volume := &ah.Volume{Number: "test-volume-number"}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedMounter.EXPECT().CreateDir(gomock.Eq(req.TargetPath)).Return(nil)
				mockedMounter.EXPECT().Mount(gomock.Eq(req.StagingTargetPath), gomock.Eq(req.TargetPath), gomock.Eq(fsType), gomock.Eq([]string{"bind", "ro"})).Return(nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodePublishVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodePublishVolume Success block",
			testFunc: func(t *testing.T) {
				req := &csi.NodePublishVolumeRequest{
					VolumeId:          "test-id",
					VolumeCapability:  volBlockCap,
					StagingTargetPath: "/staging/target/path",
					TargetPath:        "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				volume := &ah.Volume{Number: "test-volume-number"}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedMounter.EXPECT().DevicePath(gomock.Eq(volume.Number)).Return(sourcePath, nil)

				mockedMounter.EXPECT().CreateDir(gomock.Eq(filepath.Dir(req.TargetPath))).Return(nil)
				mockedMounter.EXPECT().CreateFile(gomock.Eq(req.TargetPath)).Return(nil)
				mockedMounter.EXPECT().Mount(gomock.Eq(sourcePath), gomock.Eq(req.TargetPath), gomock.Eq(""), gomock.Eq([]string{"bind"})).Return(nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodePublishVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodePublishVolume Success read only block",
			testFunc: func(t *testing.T) {
				req := &csi.NodePublishVolumeRequest{
					VolumeId:          "test-id",
					VolumeCapability:  volBlockCap,
					StagingTargetPath: "/staging/target/path",
					TargetPath:        "/target/path",
					Readonly:          true,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				volume := &ah.Volume{Number: "test-volume-number"}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedMounter.EXPECT().DevicePath(gomock.Eq(volume.Number)).Return(sourcePath, nil)

				mockedMounter.EXPECT().CreateDir(gomock.Eq(filepath.Dir(req.TargetPath))).Return(nil)
				mockedMounter.EXPECT().CreateFile(gomock.Eq(req.TargetPath)).Return(nil)
				mockedMounter.EXPECT().Mount(gomock.Eq(sourcePath), gomock.Eq(req.TargetPath), gomock.Eq(""), gomock.Eq([]string{"bind", "ro"})).Return(nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodePublishVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodePublishVolume Empty volume ID",
			testFunc: func(t *testing.T) {
				req := &csi.NodePublishVolumeRequest{
					VolumeId:          "",
					VolumeCapability:  volBlockCap,
					StagingTargetPath: "/staging/target/path",
					TargetPath:        "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodePublishVolume(ctx, req)
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
			name: "NodePublishVolume Volume not found",
			testFunc: func(t *testing.T) {
				req := &csi.NodePublishVolumeRequest{
					VolumeId:          "test-id",
					VolumeCapability:  volBlockCap,
					StagingTargetPath: "/staging/target/path",
					TargetPath:        "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(nil, ah.ErrResourceNotFound)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodePublishVolume(ctx, req)
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
			name: "NodePublishVolume Empty staging target path",
			testFunc: func(t *testing.T) {
				req := &csi.NodePublishVolumeRequest{
					VolumeId:         "test-id",
					VolumeCapability: volBlockCap,
					TargetPath:       "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodePublishVolume(ctx, req)
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
			name: "NodePublishVolume Empty target path",
			testFunc: func(t *testing.T) {
				req := &csi.NodePublishVolumeRequest{
					VolumeId:          "test-id",
					VolumeCapability:  volBlockCap,
					StagingTargetPath: "/staging/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodePublishVolume(ctx, req)
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
			name: "NodePublishVolume Empty capability",
			testFunc: func(t *testing.T) {
				req := &csi.NodePublishVolumeRequest{
					VolumeId:          "test-id",
					TargetPath:        "/target/path",
					StagingTargetPath: "/staging/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodePublishVolume(ctx, req)
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
			name: "NodePublishVolume Invalid capability",
			testFunc: func(t *testing.T) {
				volCap := &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Block{
						Block: &csi.VolumeCapability_BlockVolume{},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
					},
				}
				req := &csi.NodePublishVolumeRequest{
					VolumeId:          "test-id",
					TargetPath:        "/target/path",
					StagingTargetPath: "/staging/target/path",
					VolumeCapability:  volCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestNodeUnpublishVolume(t *testing.T) {
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "NodeUnpublishVolume Success",
			testFunc: func(t *testing.T) {
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId:   "test-id",
					TargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedMounter.EXPECT().Unmount(gomock.Eq(req.TargetPath)).Return(nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeUnpublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeUnpublishVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodeUnpublishVolume Empty volume id",
			testFunc: func(t *testing.T) {
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId:   "",
					TargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeUnpublishVolume(ctx, req)
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
			name: "NodeUnpublishVolume Volume not found",
			testFunc: func(t *testing.T) {
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId:   "test-id",
					TargetPath: "/target/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(nil, ah.ErrResourceNotFound)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeUnpublishVolume(ctx, req)
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
			name: "NodeUnpublishVolume Empty target path",
			testFunc: func(t *testing.T) {
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId: "test-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeUnpublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.InvalidArgument {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestNodeGetVolumeStats(t *testing.T) {
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "NodeGetVolumeStats Success block",
			testFunc: func(t *testing.T) {
				req := &csi.NodeGetVolumeStatsRequest{
					VolumeId:   "test-id",
					VolumePath: "/volume/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedMounter.EXPECT().IsPathExists(gomock.Eq(req.VolumePath)).Return(true, nil)
				mockedMounter.EXPECT().IsBlockDevice(gomock.Eq(req.VolumePath)).Return(true, nil)
				mockedMounter.EXPECT().BlockSize(gomock.Eq(req.VolumePath)).Return(int64(12345), nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeGetVolumeStats(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeGetVolumeStatsResponse{
					Usage: []*csi.VolumeUsage{
						{
							Unit:  csi.VolumeUsage_BYTES,
							Total: 12345,
						},
					},
				}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodeGetVolumeStats Success fs",
			testFunc: func(t *testing.T) {
				req := &csi.NodeGetVolumeStatsRequest{
					VolumeId:   "test-id",
					VolumePath: "/volume/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedMounter.EXPECT().IsPathExists(gomock.Eq(req.VolumePath)).Return(true, nil)
				mockedMounter.EXPECT().IsBlockDevice(gomock.Eq(req.VolumePath)).Return(false, nil)
				mockedMounter.EXPECT().BytesFSMetrics(gomock.Eq(req.VolumePath)).Return(int64(1), int64(2), int64(3), nil)
				mockedMounter.EXPECT().INodeFSMetrics(gomock.Eq(req.VolumePath)).Return(int64(11), int64(22), int64(33), nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeGetVolumeStats(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeGetVolumeStatsResponse{
					Usage: []*csi.VolumeUsage{
						{
							Unit:      csi.VolumeUsage_BYTES,
							Available: int64(1),
							Total:     int64(2),
							Used:      int64(3),
						},
						{
							Unit:      csi.VolumeUsage_INODES,
							Available: int64(11),
							Total:     int64(22),
							Used:      int64(33),
						},
					},
				}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodeGetVolumeStats Empty volume id",
			testFunc: func(t *testing.T) {
				req := &csi.NodeGetVolumeStatsRequest{
					VolumeId:   "",
					VolumePath: "/volume/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeGetVolumeStats(ctx, req)
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
			name: "NodeGetVolumeStats Volume not found",
			testFunc: func(t *testing.T) {
				req := &csi.NodeGetVolumeStatsRequest{
					VolumeId:   "test-id",
					VolumePath: "/volume/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(nil, ah.ErrResourceNotFound)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeGetVolumeStats(ctx, req)
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
			name: "NodeGetVolumeStats Empty volume path",
			testFunc: func(t *testing.T) {
				req := &csi.NodeGetVolumeStatsRequest{
					VolumeId: "test-id",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeGetVolumeStats(ctx, req)
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
			name: "NodeGetVolumeStats Volume path not found",
			testFunc: func(t *testing.T) {
				req := &csi.NodeGetVolumeStatsRequest{
					VolumeId:   "test-id",
					VolumePath: "/volume/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedMounter.EXPECT().IsPathExists(gomock.Eq(req.VolumePath)).Return(false, nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeGetVolumeStats(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.NotFound {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestNodeExpandVolume(t *testing.T) {
	fsType := "ext3"
	volMountCap := &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{
				FsType: fsType,
			},
		},
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		},
	}
	volBlockCap := &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Block{
			Block: &csi.VolumeCapability_BlockVolume{},
		},
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		},
	}
	devicePath := "/device/path"

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "NodeExpandVolume Success mount",
			testFunc: func(t *testing.T) {
				req := &csi.NodeExpandVolumeRequest{
					VolumeId:         "test-id",
					VolumeCapability: volMountCap,
					VolumePath:       "/volume/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedMounter.EXPECT().DeviceNameFromMount(gomock.Eq(req.VolumePath)).Return(devicePath, nil)
				mockedMounter.EXPECT().Resize(gomock.Eq(devicePath), gomock.Eq(req.VolumePath)).Return(nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeExpandVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeExpandVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodeExpandVolume Success block",
			testFunc: func(t *testing.T) {
				req := &csi.NodeExpandVolumeRequest{
					VolumeId:         "test-id",
					VolumeCapability: volBlockCap,
					VolumePath:       "/volume/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				volume := &ah.Volume{Number: "test-volume-number"}
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(volume, nil)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeExpandVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeExpandVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodeExpandVolume Empty volume",
			testFunc: func(t *testing.T) {
				req := &csi.NodeExpandVolumeRequest{
					VolumeId:         "",
					VolumeCapability: volMountCap,
					VolumePath:       "/volume/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeExpandVolume(ctx, req)
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
			name: "NodeExpandVolume Volume not found",
			testFunc: func(t *testing.T) {
				req := &csi.NodeExpandVolumeRequest{
					VolumeId:         "test-id",
					VolumeCapability: volMountCap,
					VolumePath:       "/volume/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(nil, ah.ErrResourceNotFound)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeExpandVolume(ctx, req)
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
			name: "NodeExpandVolume Empty volume path",
			testFunc: func(t *testing.T) {
				req := &csi.NodeExpandVolumeRequest{
					VolumeId:         "test-id",
					VolumeCapability: volMountCap,
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)
				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeExpandVolume(ctx, req)
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
			name: "NodeExpandVolume Empty capability",
			testFunc: func(t *testing.T) {
				req := &csi.NodeExpandVolumeRequest{
					VolumeId:   "test-id",
					VolumePath: "/volume/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedMounter.EXPECT().DeviceNameFromMount(gomock.Eq(req.VolumePath)).Return(devicePath, nil)
				mockedMounter.EXPECT().Resize(gomock.Eq(devicePath), gomock.Eq(req.VolumePath)).Return(nil)

				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				resp, err := nodeSvc.NodeExpandVolume(ctx, req)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				expectedResult := &csi.NodeExpandVolumeResponse{}
				if !reflect.DeepEqual(resp, expectedResult) {
					t.Errorf("Unexpected result, expected %v. got: %v", expectedResult, resp)
				}
			},
		},
		{
			name: "NodeExpandVolume Invalid capability",
			testFunc: func(t *testing.T) {
				volCap := &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{
							FsType: fsType,
						},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
					},
				}
				req := &csi.NodeExpandVolumeRequest{
					VolumeId:         "test-id",
					VolumeCapability: volCap,
					VolumePath:       "/volume/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeExpandVolume(ctx, req)
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
			name: "NodeExpandVolume Invalid volume path",
			testFunc: func(t *testing.T) {
				req := &csi.NodeExpandVolumeRequest{
					VolumeId:         "test-id",
					VolumeCapability: volMountCap,
					VolumePath:       "/volume/path",
				}

				ctx := context.Background()

				ctrl := gomock.NewController(t)
				mockedMounter := mocks.NewMockMounter(ctrl)
				mockedVolumesAPI := mocks.NewMockVolumesAPI(ctrl)

				mockedVolumesAPI.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(req.VolumeId)).Return(&ah.Volume{}, nil)
				mockedMounter.EXPECT().DeviceNameFromMount(gomock.Eq(req.VolumePath)).Return("", nil)
				mockedClient := &ah.APIClient{Volumes: mockedVolumesAPI}

				nodeSvc := nodeService{client: mockedClient, mounter: mockedMounter}

				r, err := nodeSvc.NodeExpandVolume(ctx, req)
				if err == nil {
					t.Fatalf("Unexpected response: %v", r)
				}

				respErr, _ := status.FromError(err)
				if respErr.Code() != codes.NotFound {
					t.Fatalf("Unexpected error: %v, %v", respErr.Code(), respErr.Message())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}
