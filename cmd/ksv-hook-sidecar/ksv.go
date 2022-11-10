/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2019 StackPath, LLC
 *
 */

// Inspired by cmd/example-hook-sidecar

package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net"
	"os"
	"path/filepath"

	virtwrapApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"google.golang.org/grpc"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	hooks "kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
)

type MyDomainSpec struct {
	virtwrapApi.DomainSpec
	Devices MyDevices `xml:"devices"`
}

type MyDevices struct {
	virtwrapApi.Devices
	Disks []MyDisk `xml:"disk"`
}

type infoServer struct{}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: "ksv-sidecar",
		Versions: []string{
			hooksV1alpha2.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			{
				Name:     hooksInfo.PreCloudInitIsoHookPointName,
				Priority: 0,
			},
			{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type v1alpha2Server struct{}

func (s v1alpha2Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha2.OnDefineDomainParams) (*hooksV1alpha2.OnDefineDomainResult, error) {
	log.Log.Warning("Hook's OnDefineDomain callback method has been called")

	// vmi convert
	vmiJSON := params.GetVmi()
	vmi := v1.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmi)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given VMI spec: %s", vmiJSON)
		return &hooksV1alpha2.OnDefineDomainResult{
			DomainXML: params.GetDomainXML(),
		}, nil
	}

	// check if here needs to continue base on vmi annotations
	if !checkContinue(&vmi) {
		log.Log.Info("Don't need to adjust domain xml")
		return &hooksV1alpha2.OnDefineDomainResult{
			DomainXML: params.GetDomainXML(),
		}, nil
	}

	// try to get the iotune parameter from annotations
	myIoTune := gotMyIoTuneParam(vmi.Annotations)

	domainXML := params.GetDomainXML()
	myDomainSpec := MyDomainSpec{}
	err = xml.Unmarshal(domainXML, &myDomainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", domainXML)
		return &hooksV1alpha2.OnDefineDomainResult{
			DomainXML: params.GetDomainXML(),
		}, nil
	}

	// modify domain spec xml for disk
	for i, myDisk := range myDomainSpec.Devices.Disks {
		adjustDiskAttributes(myIoTune, &myDisk)
		myDomainSpec.Devices.Disks[i] = myDisk
	}

	marshal, err := xml.Marshal(myDomainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to Marshal updated domain spec: %v", myDomainSpec)
		return &hooksV1alpha2.OnDefineDomainResult{
			DomainXML: params.GetDomainXML(),
		}, nil
	}

	log.Log.Infof("updated before domain xml: %s", string(domainXML))
	log.Log.Infof("updated after  domain xml: %s", string(marshal))

	return &hooksV1alpha2.OnDefineDomainResult{
		DomainXML: marshal,
	}, nil
}

func (s v1alpha2Server) PreCloudInitIso(ctx context.Context, params *hooksV1alpha2.PreCloudInitIsoParams) (*hooksV1alpha2.PreCloudInitIsoResult, error) {
	log.Log.Info("Hook's PreCloudInitIso callback method has been called")

	cloudInitDataJSON := params.GetCloudInitData()
	cloudInitData := cloudinit.CloudInitData{}
	err := json.Unmarshal(cloudInitDataJSON, &cloudInitData)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given CloudInitData: %s", cloudInitDataJSON)
		panic(err)
	}

	log.Log.Infof("Hook's PreCloudInitIso NetworkData: %s", cloudInitData.NetworkData)
	log.Log.Infof("Hook's PreCloudInitIso UserData: %s", cloudInitData.UserData)

	return &hooksV1alpha2.PreCloudInitIsoResult{
		CloudInitData: cloudInitDataJSON,
	}, nil
}

func checkContinue(vmi *v1.VirtualMachineInstance) bool {
	if checkContinueForIoTune(vmi) {
		return true
	}
	return false
}

func main() {
	log.InitializeLogging("ksv-hook-sidecar")

	socketPath := filepath.Join(hooks.HookSocketsSharedDirectory, "ksv-hook-sidecar.sock")
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)
	hooksInfo.RegisterInfoServer(server, infoServer{})
	hooksV1alpha2.RegisterCallbacksServer(server, v1alpha2Server{})
	log.Log.Infof("Starting ksv hook sidecar server exposing 'info' and 'v1alpha2' services on socket %s", socketPath)
	server.Serve(socket)
}
