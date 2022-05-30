/*
 * This file is part of the KubeVirt project
 *
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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	vmSchema "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	hostUsbDeviceAnnotation      = "host.usb.vm.kubevirt.io/device" //vendor:product (0x0951:0x1665)
	hostUsbControllerAnnotation  = "host.usb.vm.kubevirt.io/controller"
	onDefineDomainLoggingMessage = "Host use device hook's OnDefineDomain callback method has been called"
)

type infoServer struct {
	Version string
}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: "hostusb",
		Versions: []string{
			s.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type v1alpha1Server struct{}
type v1alpha2Server struct{}

func (s v1alpha2Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha2.OnDefineDomainParams) (*hooksV1alpha2.OnDefineDomainResult, error) {
	log.Log.Info(onDefineDomainLoggingMessage)
	newDomainXML, err := onDefineDomain(params.GetVmi(), params.GetDomainXML())
	if err != nil {
		return nil, err
	}
	return &hooksV1alpha2.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}
func (s v1alpha2Server) PreCloudInitIso(_ context.Context, params *hooksV1alpha2.PreCloudInitIsoParams) (*hooksV1alpha2.PreCloudInitIsoResult, error) {
	return &hooksV1alpha2.PreCloudInitIsoResult{
		CloudInitData: params.GetCloudInitData(),
	}, nil
}

func (s v1alpha1Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha1.OnDefineDomainParams) (*hooksV1alpha1.OnDefineDomainResult, error) {
	log.Log.Info(onDefineDomainLoggingMessage)
	newDomainXML, err := onDefineDomain(params.GetVmi(), params.GetDomainXML())
	if err != nil {
		return nil, err
	}
	return &hooksV1alpha1.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func onDefineDomain(vmiJSON []byte, domainXML []byte) ([]byte, error) {
	log.Log.Info(onDefineDomainLoggingMessage)

	vmiSpec := vmSchema.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmiSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given VMI spec: %s", vmiJSON)
		panic(err)
	}

	annotations := vmiSpec.GetAnnotations()

	device, found := annotations[hostUsbDeviceAnnotation]
	if !found {
		log.Log.Info("Host usb device hook sidecar was requested, but no attributes provided. Returning original domain spec")
		return domainXML, nil
	}
	split := strings.Split(device, ":")
	if len(split) != 2 {
		log.Log.Info("Host usb device hook sidecar was requested, but don't match format vendor:product (0x0951:0x1665). Returning original domain spec")
		return domainXML, nil
	}

	domainSpec := domainSchema.DomainSpec{}
	err = xml.Unmarshal(domainXML, &domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", domainXML)
		panic(err)
	}

	ctlModel := annotations[hostUsbControllerAnnotation]
	if ctlModel == "" {
		ctlModel = "piix3-uhci"
	}

	// usb controller
	domainSpec.Devices.Controllers = append(domainSpec.Devices.Controllers, domainSchema.Controller{
		Type:  "usb",
		Model: ctlModel,
		Index: "0",
	})

	// host usb device
	domainSpec.Devices.HostDevices = append(domainSpec.Devices.HostDevices, domainSchema.HostDevice{
		Mode:    "subsystem",
		Type:    "usb",
		Managed: "yes",
		Source: domainSchema.HostDeviceSource{
			Vendor: domainSchema.ID{
				Id: split[0],
			},
			Product: domainSchema.ID{
				Id: split[1],
			},
		},
		Address: &domainSchema.Address{
			Type: "usb",
			Bus:  "0",
			Port: "1",
		},
	})

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal updated domain spec: %+v", domainSpec)
		panic(err)
	}

	log.Log.Info("Successfully updated original domain spec with requested host usb devices")

	return newDomainXML, nil
}

func main() {
	log.InitializeLogging("host-usb-hook-sidecar")

	var version string
	pflag.StringVar(&version, "version", "v1alpha2", "hook version to use")
	pflag.Parse()

	socketPath := filepath.Join(hooks.HookSocketsSharedDirectory, "host-usb.sock")
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)

	if version == "" {
		panic(fmt.Errorf("usage: \n        /example-hook-sidecar --version v1alpha1|v1alpha2"))
	}
	hooksInfo.RegisterInfoServer(server, infoServer{Version: version})
	hooksV1alpha1.RegisterCallbacksServer(server, v1alpha1Server{})
	hooksV1alpha2.RegisterCallbacksServer(server, v1alpha2Server{})
	log.Log.Infof("Starting hook server exposing 'info' and 'v1alpha1' services on socket %s", socketPath)
	server.Serve(socket)
}
