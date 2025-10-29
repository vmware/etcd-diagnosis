// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package report

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vmware/etcd-diagnosis/commands/report/agent"
	"github.com/vmware/etcd-diagnosis/commands/report/engine"
	"github.com/vmware/etcd-diagnosis/commands/report/engine/intf"
	"github.com/vmware/etcd-diagnosis/commands/report/plugins/epstatus"
	"github.com/vmware/etcd-diagnosis/commands/report/plugins/membership"
	"github.com/vmware/etcd-diagnosis/commands/report/plugins/metrics"
	"github.com/vmware/etcd-diagnosis/commands/report/plugins/read"
)

var (
	globalCfg      = agent.GlobalConfig{}
	reportFilename string
)

func NewCommandReport() *cobra.Command {
	diagnosisCmd := &cobra.Command{
		Use:   "report",
		Short: "report generates a diagnostic report for a running etcd cluster.",
		Run:   reportCommandFunc,
	}

	diagnosisCmd.Flags().StringSliceVar(&globalCfg.Endpoints, "endpoints", []string{"127.0.0.1:2379"}, "comma separated etcd endpoints")
	diagnosisCmd.Flags().BoolVar(&globalCfg.UseClusterEndpoints, "cluster", false, "use all endpoints from the cluster member list")

	diagnosisCmd.Flags().DurationVar(&globalCfg.DialTimeout, "dial-timeout", 2*time.Second, "dial timeout for client connections")
	diagnosisCmd.Flags().DurationVar(&globalCfg.CommandTimeout, "command-timeout", 5*time.Second, "command timeout (excluding dial timeout)")
	diagnosisCmd.Flags().DurationVar(&globalCfg.KeepAliveTime, "keepalive-time", 2*time.Second, "keepalive time for client connections")
	diagnosisCmd.Flags().DurationVar(&globalCfg.KeepAliveTimeout, "keepalive-timeout", 5*time.Second, "keepalive timeout for client connections")

	diagnosisCmd.Flags().BoolVar(&globalCfg.Insecure, "insecure-transport", true, "disable transport security for client connections")

	diagnosisCmd.Flags().BoolVar(&globalCfg.InsecureSkipVerify, "insecure-skip-tls-verify", false, "skip server certificate verification (CAUTION: this option should be enabled only for testing purposes)")
	diagnosisCmd.Flags().StringVar(&globalCfg.CertFile, "cert", "", "identify secure client using this TLS certificate file")
	diagnosisCmd.Flags().StringVar(&globalCfg.KeyFile, "key", "", "identify secure client using this TLS key file")
	diagnosisCmd.Flags().StringVar(&globalCfg.CaFile, "cacert", "", "verify certificates of TLS-enabled secure servers using this CA bundle")

	diagnosisCmd.Flags().StringVar(&globalCfg.Username, "user", "", "username[:password] for authentication (prompt if password is not supplied)")
	diagnosisCmd.Flags().StringVar(&globalCfg.Password, "password", "", "password for authentication (if this option is used, --user option shouldn't include password)")

	diagnosisCmd.Flags().StringVarP(&globalCfg.DNSDomain, "discovery-srv", "d", "", "domain name to query for SRV records describing cluster endpoints")
	diagnosisCmd.Flags().StringVarP(&globalCfg.DNSService, "discovery-srv-name", "", "", "service name to query when using DNS discovery")
	diagnosisCmd.Flags().BoolVar(&globalCfg.InsecureDiscovery, "insecure-discovery", true, "accept insecure SRV records describing cluster endpoints")

	diagnosisCmd.Flags().IntVar(&globalCfg.DbQuotaBytes, "etcd-storage-quota-bytes", 2*1024*1024*1024, "etcd storage quota in bytes (the value passed to etcd instance by flag --quota-backend-bytes)")

	diagnosisCmd.Flags().StringVar(&reportFilename, "output", engine.DefaultReportFileName, "file path to write the online diagnosis report")

	return diagnosisCmd
}

func reportCommandFunc(_ *cobra.Command, _ []string) {
	log.Println("etcd online diagnosis starting...")

	autoPopulateCertificate(&globalCfg)

	plugins := []intf.Plugin{
		membership.NewPlugin(globalCfg),
		epstatus.NewPlugin(globalCfg),
		read.NewPlugin(globalCfg, false),
		read.NewPlugin(globalCfg, true),
		metrics.NewPlugin(globalCfg),
	}

	engine.Diagnose(globalCfg, plugins, reportFilename)

	log.Println("etcd online diagnosis done!")
}

func autoPopulateCertificate(cfg *agent.GlobalConfig) {
	if cfg == nil || len(cfg.Endpoints) == 0 {
		return
	}

	ep := strings.ToLower(cfg.Endpoints[0])
	if ep == "" || !strings.HasPrefix(ep, "https") {
		return
	}

	if cfg.CaFile != "" || cfg.CertFile != "" || cfg.KeyFile != "" {
		return
	}

	cfg.CaFile = "/etc/kubernetes/pki/etcd/ca.crt"
	cfg.CertFile = "/etc/kubernetes/pki/etcd/healthcheck-client.crt"
	cfg.KeyFile = "/etc/kubernetes/pki/etcd/healthcheck-client.key"
	log.Printf("Use the default cert/key files, CA: %s, Cert: %s, Key: %s\n", cfg.CaFile, cfg.CertFile, cfg.KeyFile)
}
