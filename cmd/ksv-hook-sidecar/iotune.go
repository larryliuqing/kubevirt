package main

import (
	"encoding/json"
	"fmt"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	virtwrapApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"strconv"
)

type DiskIOTune struct {
	TotalBytesSec int `xml:"total_bytes_sec,omitempty"`
	ReadBytesSec  int `xml:"read_bytes_sec,omitempty"`
	WriteBytesSec int `xml:"write_bytes_sec,omitempty"`
	TotalIopsSec  int `xml:"total_iops_sec,omitempty"`
	ReadIopsSec   int `xml:"read_iops_sec,omitempty"`
	WriteIopsSec  int `xml:"write_iops_sec,omitempty"`

	TotalBytesSecMax int `xml:"total_bytes_sec_max,omitempty"`
	ReadBytesSecMax  int `xml:"read_bytes_sec_max,omitempty"`
	WriteBytesSecMax int `xml:"write_bytes_sec_max,omitempty"`
	TotalIopsSecMax  int `xml:"total_iops_sec_max,omitempty"`
	ReadIopsSecMax   int `xml:"read_iops_sec_max,omitempty"`
	WriteIopsSecMax  int `xml:"write_iops_sec_max,omitempty"`

	SizeIopsSec int    `xml:"size_iops_sec,omitempty"`
	GroupName   string `xml:"group_name,omitempty"`

	TotalBytesSecMaxLen int `xml:"total_bytes_sec_max_length,omitempty"`
	ReadBytesSecMaxLen  int `xml:"read_bytes_sec_max_length,omitempty"`
	WriteBytesSecMaxLen int `xml:"write_bytes_sec_max_length,omitempty"`
	TotalIopsSecMaxLen  int `xml:"total_iops_sec_max_length,omitempty"`
	ReadIopsSecMaxLen   int `xml:"read_iops_sec_max_length,omitempty"`
	WriteIopsSecMaxLen  int `xml:"write_iops_sec_max_length,omitempty"`
}

const DiskIoTuneAnno = "iotune.kubesphere.io/disk"

type MyDisk struct {
	virtwrapApi.Disk
	IoTune *DiskIOTune `xml:"iotune,omitempty"`
}

func checkContinueForIoTune(vmi *v1.VirtualMachineInstance) bool {
	_, ok := vmi.Annotations[DiskIoTuneAnno]
	return ok
}

func gotMyIoTuneParam(annotations map[string]string) map[string]string {
	tuneAnno := annotations[DiskIoTuneAnno]
	if tuneAnno == "" {
		return nil
	}

	myIoTune := make(map[string]string)
	err := json.Unmarshal([]byte(tuneAnno), &myIoTune)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		log.Log.Reason(err).Errorf("Unmarshal failed for %s", tuneAnno)
		return nil
	}
	return myIoTune
}

func adjustDiskAttributes(myIoTune map[string]string, myDisk *MyDisk) {
	if myIoTune == nil {
		return
	}

	myDisk.IoTune = &DiskIOTune{}
	for k, v := range myIoTune {
		switch k {
		case "total_bytes_sec":
			updateToInt(v, &myDisk.IoTune.TotalBytesSec)
		case "read_bytes_sec":
			updateToInt(v, &myDisk.IoTune.ReadBytesSec)
		case "write_bytes_sec":
			updateToInt(v, &myDisk.IoTune.WriteBytesSec)
		case "total_iops_sec":
			updateToInt(v, &myDisk.IoTune.TotalIopsSec)
		case "read_iops_sec":
			updateToInt(v, &myDisk.IoTune.ReadIopsSec)
		case "write_iops_sec":
			updateToInt(v, &myDisk.IoTune.WriteIopsSec)

		case "total_bytes_sec_max":
			updateToInt(v, &myDisk.IoTune.TotalBytesSecMax)
		case "read_bytes_sec_max":
			updateToInt(v, &myDisk.IoTune.ReadBytesSecMax)
		case "write_bytes_sec_max":
			updateToInt(v, &myDisk.IoTune.WriteBytesSecMax)
		case "total_iops_sec_max":
			updateToInt(v, &myDisk.IoTune.TotalIopsSecMax)
		case "read_iops_sec_max":
			updateToInt(v, &myDisk.IoTune.ReadIopsSecMax)
		case "write_iops_sec_max":
			updateToInt(v, &myDisk.IoTune.WriteIopsSecMax)

		case "size_iops_sec":
			updateToInt(v, &myDisk.IoTune.SizeIopsSec)
		case "group_name":
			updateToString(v, &myDisk.IoTune.GroupName)

		case "total_bytes_sec_max_length":
			updateToInt(v, &myDisk.IoTune.TotalBytesSecMaxLen)
		case "read_bytes_sec_max_length":
			updateToInt(v, &myDisk.IoTune.ReadBytesSecMaxLen)
		case "write_bytes_sec_max_length":
			updateToInt(v, &myDisk.IoTune.WriteBytesSecMaxLen)
		case "total_iops_sec_max_length":
			updateToInt(v, &myDisk.IoTune.TotalIopsSecMaxLen)
		case "read_iops_sec_max_length":
			updateToInt(v, &myDisk.IoTune.ReadIopsSecMaxLen)
		case "write_iops_sec_max_length":
			updateToInt(v, &myDisk.IoTune.WriteIopsSecMaxLen)
		}
	}
}

func updateToString(para string, res *string) {
	if para == "" {
		return
	}
	*res = para
}

func updateToInt(para string, res *int) {
	if para == "" {
		return
	}

	value, err := strconv.Atoi(para)
	if err == nil {
		*res = value
	}
}
