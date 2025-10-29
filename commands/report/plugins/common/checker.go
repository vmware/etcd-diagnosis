// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package common

import "github.com/vmware/etcd-diagnosis/commands/report/agent"

type Checker struct {
	agent.GlobalConfig
	Name string
}
