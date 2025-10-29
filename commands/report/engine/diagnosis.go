// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package engine

import (
	"encoding/json"
	"log"
	"os"

	"github.com/vmware/etcd-diagnosis/commands/report/engine/intf"
)

const (
	DefaultReportFileName = "etcd_diagnosis_report.json"
)

type report struct {
	Input   any   `json:"input,omitempty"`
	Results []any `json:"results,omitempty"`
}

func Diagnose(input any, plugins []intf.Plugin, reportFilename string) {
	rp := report{
		Input: input,
	}
	for i, plugin := range plugins {
		log.Println("---------------------------------------------------------")
		log.Printf("Running %q (%d/%d)...\n", plugin.Name(), i+1, len(plugins))

		result := plugin.Diagnose()
		rp.Results = append(rp.Results, result)

		b, err := json.MarshalIndent(result, "", "\t")
		if err != nil {
			log.Printf("Failed to marshal result for plugin %q: %v", plugin.Name(), err)
			continue
		}
		log.Println(string(b))
	}

	b, err := json.MarshalIndent(rp, "", "\t")
	if err != nil {
		log.Fatalf("Failed to marshal the report: %v", err)
	}

	if reportFilename == "" {
		reportFilename = DefaultReportFileName
	}
	if err := os.WriteFile(reportFilename, b, 0o644); err != nil {
		log.Fatalf("Failed to write the report to file: %v", err)
	}
}
