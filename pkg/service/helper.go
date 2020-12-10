package service

import (
	"fmt"

	ovirtsdk "github.com/ovirt/go-ovirt"
)

func diskAttachmentByVmAndDisk(connection *ovirtsdk.Connection, vmId string, diskId string) (*ovirtsdk.DiskAttachment, error) {
	vmService := connection.SystemService().VmsService().VmService(vmId)
	attachments, err := vmService.DiskAttachmentsService().List().Send()
	if err != nil {
		return nil, err
	}

	for _, attachment := range attachments.MustAttachments().Slice() {
		if diskId == attachment.MustDisk().MustId() {
			return attachment, nil
		}
	}
	return nil, fmt.Errorf("failed to find attachment by disk %s for VM %s", diskId, vmId)
}

func diskAttachmentByDisk(connection *ovirtsdk.Connection, diskId string) (*ovirtsdk.DiskAttachment, *ovirtsdk.DiskAttachmentService, error) {
	vmsService := connection.SystemService().VmsService()
	for _, vm := range vmsService.List().MustSend().MustVms().Slice() {
		vmService := connection.SystemService().VmsService().VmService(vm.MustId())
		attachments, err := vmService.DiskAttachmentsService().List().Send()
		if err != nil {
			return nil, nil, err
		}

		for _, attachment := range attachments.MustAttachments().Slice() {
			if attachment.MustId() == diskId {
				return attachment, vmService.DiskAttachmentsService().AttachmentService(attachment.MustId()), nil

			}
		}
	}
	return nil, nil, fmt.Errorf("failed to find diskAttachment and diskAttachmentService by disk %s", diskId)
}