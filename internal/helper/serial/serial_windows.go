package serial

import (
	"fmt"

	"github.com/microsoft/wmi/pkg/base/host"
	"github.com/microsoft/wmi/pkg/base/instance"
	"github.com/microsoft/wmi/pkg/base/query"
	"github.com/microsoft/wmi/pkg/constant"
	"github.com/microsoft/wmi/server2019/root/cimv2"
)

func GetDeviceSerial() (string, error) {
	biosInstance, err := instance.GetWmiInstanceEx(host.NewWmiLocalHost(), string(constant.CimV2), query.NewWmiQuery("Win32_BIOS"))
	if err != nil {
		return "", fmt.Errorf("getting wmi instance: %w", err)
	}

	bios, err := cimv2.NewCIM_BIOSElementEx1(biosInstance)
	if err != nil {
		return "", fmt.Errorf("getting bios element: %w", err)
	}

	serial, err := bios.GetPropertySerialNumber()
	if err != nil {
		return "", fmt.Errorf("getting serial number: %w", err)
	}

	return serial, nil
}
