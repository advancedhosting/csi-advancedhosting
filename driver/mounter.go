package driver

import (
	"fmt"
	"golang.org/x/sys/unix"
	"k8s.io/kubernetes/pkg/util/resizefs"
	"k8s.io/kubernetes/pkg/volume"
	"k8s.io/utils/exec"
	"k8s.io/utils/mount"
	"os"
	"strconv"
	"strings"
)

type Mounter interface {
	DevicePath(string) (string, error)
	IsPathExists(string) (bool, error)
	FormatAndMount(string, string, string, []string) error
	Mount(string, string, string, []string) error
	Unmount(string) error
	CreateDir(string) error
	CreateFile(string) error
	DeviceNameFromMount(string) (string, error)
	IsDeviceMountedToTarget(string, string) (bool, error)
	IsBlockDevice(string) (bool, error)
	BlockSize(string) (int64, error)
	BytesFSMetrics(string) (int64, int64, int64, error)
	INodeFSMetrics(string) (int64, int64, int64, error)
	Resize(string, string) error
}

type LinuxMounter struct {
	mounter *mount.SafeFormatAndMount
}

func NewLinuxMounter() *LinuxMounter {
	return &LinuxMounter{
		mounter: &mount.SafeFormatAndMount{
			Interface: mount.New(""),
			Exec:      exec.New(),
		},
	}
}

var _ Mounter = (*LinuxMounter)(nil)

func (m *LinuxMounter) DevicePath(volumeNumber string) (string, error) {
	devicePath := fmt.Sprintf("/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_%s", volumeNumber)
	exists, err := m.IsPathExists(devicePath)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("device %s not found", devicePath)
	}
	return devicePath, nil
}

func (m *LinuxMounter) IsPathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (m *LinuxMounter) FormatAndMount(source, target, fsType string, options []string) error {
	return m.mounter.FormatAndMount(source, target, fsType, options)
}

func (m *LinuxMounter) Mount(source, target, fsType string, options []string) error {
	return m.mounter.Mount(source, target, "", options)
}

func (m *LinuxMounter) CreateDir(path string) error {
	err := os.MkdirAll(path, 0750)
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create directory %s: %v", path, err)
		}
	}
	return nil
}

func (m *LinuxMounter) CreateFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE, os.FileMode(0644))
	if err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return nil
}

func (m *LinuxMounter) IsDeviceMountedToTarget(devicePath, targetPath string) (bool, error) {
	device, err := m.DeviceNameFromMount(targetPath)
	if err != nil {
		return false, err
	}
	return device == devicePath, nil
}

func (m *LinuxMounter) DeviceNameFromMount(mountPath string) (string, error) {
	device, _, err := mount.GetDeviceNameFromMount(m.mounter, mountPath)
	if err != nil {
		return "", err
	}
	return device, nil
}

func (m *LinuxMounter) Unmount(target string) error {
	return m.mounter.Unmount(target)
}

func (m *LinuxMounter) IsBlockDevice(path string) (bool, error) {
	var stat unix.Stat_t
	err := unix.Stat(path, &stat)
	if err != nil {
		return false, err
	}
	return (stat.Mode & unix.S_IFMT) == unix.S_IFBLK, nil
}

func (m *LinuxMounter) BlockSize(path string) (int64, error) {
	cmd := m.mounter.Exec.Command("blockdev", "--getsize64", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	strOut := strings.TrimSpace(string(output))
	gotSizeBytes, err := strconv.ParseInt(strOut, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size %s as int", strOut)
	}
	return gotSizeBytes, nil
}

func (m *LinuxMounter) BytesFSMetrics(path string) (available, total, used int64, err error) {
	metricsProvider := volume.NewMetricsStatFS(path)

	metrics, err := metricsProvider.GetMetrics()
	if err != nil {
		return
	}

	available = metrics.Available.AsDec().UnscaledBig().Int64()
	total = metrics.Capacity.AsDec().UnscaledBig().Int64()
	used = metrics.Used.AsDec().UnscaledBig().Int64()
	return
}

func (m *LinuxMounter) INodeFSMetrics(path string) (available, total, used int64, err error) {
	metricsProvider := volume.NewMetricsStatFS(path)

	metrics, err := metricsProvider.GetMetrics()
	if err != nil {
		return
	}

	available = metrics.InodesFree.AsDec().UnscaledBig().Int64()
	total = metrics.Inodes.AsDec().UnscaledBig().Int64()
	used = metrics.InodesUsed.AsDec().UnscaledBig().Int64()
	return
}

func (m *LinuxMounter) Resize(devicePath, deviceMountPath string) error {
	resizer := resizefs.NewResizeFs(m.mounter)
	if _, err := resizer.Resize(devicePath, deviceMountPath); err != nil {
		return err
	}
	return nil
}
