package main

import (
	"encoding/json"
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	virtwrapApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type BandWidth struct {
	Inbound  *Inbound  `xml:"inbound,omitempty"`
	Outbound *Outbound `xml:"outbound,omitempty"`
}

type Inbound struct {
	Average string `xml:"average,attr"`
	Peak    string `xml:"peak,attr,omitempty"`
	Burst   string `xml:"burst,attr,omitempty"`
	Floor   string `xml:"floor,attr,omitempty"`
}

type Outbound struct {
	Average string `xml:"average,attr"`
	Peak    string `xml:"peak,attr,omitempty"`
	Burst   string `xml:"burst,attr,omitempty"`
}

const IfaceTuneAnno = "ifacetune.kubesphere.io/bandwidth"

type MyInterface struct {
	virtwrapApi.Interface
	BandWidth *BandWidth `xml:"bandwidth,omitempty"`
}

type MyIfaceTuneAnno struct {
	Inbound  map[string]string `json:"inbound,omitempty"`
	Outbound map[string]string `json:"outbound,omitempty"`
}

func checkContinueForIfTune(vmi *v1.VirtualMachineInstance) bool {
	_, ok := vmi.Annotations[IfaceTuneAnno]
	return ok
}

func gotMyIfTuneParam(annotations map[string]string) map[string]*MyIfaceTuneAnno {
	tuneAnno := annotations[IfaceTuneAnno]
	if tuneAnno == "" {
		return nil
	}

	myIfaceTuneAnno := make(map[string]*MyIfaceTuneAnno)
	err := json.Unmarshal([]byte(tuneAnno), &myIfaceTuneAnno)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		log.Log.Reason(err).Errorf("Unmarshal failed for %s", tuneAnno)
		return nil
	}
	return myIfaceTuneAnno
}

func adjustInterfaceAttributes(myIfaceTuneAnno map[string]*MyIfaceTuneAnno, myIface *MyInterface) {
	if myIface.MAC == nil || myIface.MAC.MAC == "" {
		return
	}

	tuneAnno := myIfaceTuneAnno[myIface.MAC.MAC]
	if tuneAnno == nil {
		return
	}

	if myIface.BandWidth == nil {
		myIface.BandWidth = &BandWidth{}
	}

	// process for inbound
	for k, v := range tuneAnno.Inbound {
		if myIface.BandWidth.Inbound == nil {
			myIface.BandWidth.Inbound = &Inbound{}
		}
		switch k {
		case "average":
			myIface.BandWidth.Inbound.Average = v
		case "peak":
			myIface.BandWidth.Inbound.Peak = v
		case "burst":
			myIface.BandWidth.Inbound.Burst = v
		case "floor":
			myIface.BandWidth.Inbound.Floor = v
		}
	}

	// process for outbound
	for k, v := range tuneAnno.Outbound {
		if myIface.BandWidth.Outbound == nil {
			myIface.BandWidth.Outbound = &Outbound{}
		}
		switch k {
		case "average":
			myIface.BandWidth.Outbound.Average = v
		case "peak":
			myIface.BandWidth.Outbound.Peak = v
		case "burst":
			myIface.BandWidth.Outbound.Burst = v
		}
	}
}
