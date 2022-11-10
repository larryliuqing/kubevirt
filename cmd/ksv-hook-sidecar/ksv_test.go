package main

import (
	"context"
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
)

func Test_v1alpha2Server_OnDefineDomain(t *testing.T) {
	var domainXml = []byte(`<domain type='kvm' id='1' xmlns:qemu='http://libvirt.org/schemas/domain/qemu/1.0'>
  <name>default_i-js7isze7</name>
  <uuid>3e4690eb-a343-5d13-a960-8e506f025cac</uuid>
  <memory unit='KiB'>1048576</memory>
  <currentMemory unit='KiB'>1048576</currentMemory>
  <vcpu placement='static'>1</vcpu>
  <clock offset='utc'/>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>destroy</on_crash>
  <devices>
    <emulator>/usr/libexec/qemu-kvm</emulator>
    <disk type='block' device='disk' model='virtio-non-transitional'>
      <driver name='qemu' type='raw' cache='none' error_policy='stop' io='native' discard='unmap'/>
      <source dev='/dev/vol-5kjkm50i' index='2'/>
      <backingStore/>
      <target dev='vda' bus='virtio'/>
      <boot order='1'/>
      <alias name='ua-vol-5kjkm50i'/>
      <address type='pci' domain='0x0000' bus='0x04' slot='0x00' function='0x0'/>
    </disk>
    <disk type='file' device='disk' model='virtio-non-transitional'>
      <driver name='qemu' type='raw' cache='none' error_policy='stop' discard='unmap'/>
      <source file='/var/run/kubevirt-ephemeral-disks/cloud-init-data/default/i-js7isze7/noCloud.iso' index='1'/>
      <backingStore/>
      <target dev='vdb' bus='virtio'/>
      <alias name='ua-cloudinitdisk'/>
      <address type='pci' domain='0x0000' bus='0x05' slot='0x00' function='0x0'/>
    </disk>
    <disk type='block' device='disk'>
      <driver name='qemu' type='raw' error_policy='stop' discard='unmap'/>
      <source dev='/var/run/kubevirt/hotplug-disks/vol-73ub9mbs' index='3'/>
      <backingStore/>
      <target dev='sda' bus='scsi'/>
      <alias name='ua-vol-73ub9mbs'/>
      <address type='drive' controller='0' bus='0' target='0' unit='0'/>
    </disk>
    <disk type='block' device='disk'>
      <driver name='qemu' type='raw' error_policy='stop' discard='unmap'/>
      <source dev='/var/run/kubevirt/hotplug-disks/vol-idqoa2m1' index='4'/>
      <backingStore/>
      <target dev='sdb' bus='scsi'/>
      <alias name='ua-vol-idqoa2m1'/>
      <address type='drive' controller='0' bus='0' target='0' unit='1'/>
    </disk>
    <interface type='ethernet'>
      <mac address='86:5d:c0:a8:64:dd'/>
      <target dev='net1' managed='no'/>
      <model type='virtio-non-transitional'/>
      <mtu size='1500'/>
      <alias name='ua-eth0'/>
      <rom enabled='no'/>
      <address type='pci' domain='0x0000' bus='0x01' slot='0x00' function='0x0'/>
    </interface>
    <controller type='usb' index='0' model='none'>
      <alias name='usb'/>
    </controller>
  </devices>
</domain>`)

	vmiObj := v1.VirtualMachineInstance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachineInstance",
			APIVersion: "kubevirt.io/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "i-js7isze7",
			Annotations: map[string]string{
				"iotune.kubesphere.io/disk": "{\"read_bytes_sec\":\"10485760\",\"read_iops_sec\":\"1000\",\"write_bytes_sec\":\"10485760\",\"write_iops_sec\":\"1000\"}",
			},
		},
		Spec: v1.VirtualMachineInstanceSpec{},
	}

	vmiJson, _ := json.Marshal(vmiObj)
	ctx := context.Background()
	s := v1alpha2Server{}
	_, err := s.OnDefineDomain(ctx, &hooksV1alpha2.OnDefineDomainParams{
		DomainXML: domainXml,
		Vmi:       vmiJson,
	})

	if err != nil {
		t.Errorf("OnDefineDomain() error = %v", err)
		return
	}
}
