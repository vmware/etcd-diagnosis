// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package metrics

import (
	"fmt"
	"log"
	"math"
	"time"

	dto "github.com/prometheus/client_model/go"

	"github.com/vmware/etcd-diagnosis/commands/report/agent"
	"github.com/vmware/etcd-diagnosis/commands/report/engine/intf"
	"github.com/vmware/etcd-diagnosis/commands/report/plugins/common"
)

var (
	metricsNames = []string{
		"etcd_disk_wal_fsync_duration_seconds",
		"etcd_disk_backend_commit_duration_seconds",
		"etcd_network_peer_round_trip_time_seconds",
		"process_resident_memory_bytes",
		//"process_cpu_seconds_total",
	}
	percentileOptions = []int{
		99, 95, 90, 85, 80, 75, 50,
	}
)

type metricsChecker struct {
	common.Checker
}

type checkResult struct {
	Name          string      `json:"name,omitempty"`
	Summary       []string    `json:"summary,omitempty"`
	EpMetricsList []epMetrics `json:"epMetricsList,omitempty"`
}

type epMetrics struct {
	Endpoint  string                  `json:"endpoint,omitempty"`
	Took      string                  `json:"took,omitempty"`
	EpMetrics map[string]MetricOutput `json:"epMetrics,omitempty"`
}

type MetricOutput struct {
	Type    string         `json:"type"`
	Help    string         `json:"help"`
	Samples []MetricSample `json:"samples"`
}

type MetricSample struct {
	Labels map[string]string `json:"labels,omitempty"`
	Metric MetricValue       `json:"metric"`
}

type MetricValue any

type GaugeMetric struct {
	Value float64 `json:"value"`
}

type HistogramMetric struct {
	Count       uint64            `json:"count"`
	Sum         float64           `json:"sum"`
	Buckets     []HistogramBucket `json:"buckets,omitempty"`
	Percentiles []Percentile      `json:"percentiles,omitempty"`
}

type HistogramBucket struct {
	Le    SafeFloat64 `json:"le"`
	Count uint64      `json:"count"`
}

type Percentile struct {
	Name  int         `json:"percentile"`
	Value SafeFloat64 `json:"value"`
}

type SafeFloat64 float64

func (f SafeFloat64) MarshalJSON() ([]byte, error) {
	v := float64(f)
	switch {
	case math.IsInf(v, 1):
		return []byte(`"Inf"`), nil
	case math.IsInf(v, -1):
		return []byte(`"-Inf"`), nil
	case math.IsNaN(v):
		return []byte(`"NaN"`), nil
	default:
		return []byte(fmt.Sprintf("%g", v)), nil
	}
}

func NewPlugin(gcfg agent.GlobalConfig) intf.Plugin {
	return &metricsChecker{
		Checker: common.Checker{
			GlobalConfig: gcfg,
			Name:         "metricsChecker",
		},
	}
}

func (ck *metricsChecker) Name() string {
	return ck.Checker.Name
}

func (ck *metricsChecker) Diagnose() (result any) {
	var (
		eps []string
		err error
	)

	defer func() {
		if err != nil {
			result = &intf.FailedResult{
				Name:   ck.Name(),
				Reason: err.Error(),
			}
		}
	}()

	if eps, err = agent.Endpoints(ck.GlobalConfig); err != nil {
		log.Printf("Failed to get endpoint: %v\n", err)
		return result
	}
	log.Printf("Endpoints: %v\n", eps)

	chkResult := checkResult{
		Name:          ck.Name(),
		Summary:       []string{},
		EpMetricsList: make([]epMetrics, len(eps)),
	}

	for i, ep := range eps {
		chkResult.EpMetricsList[i].Endpoint = ep

		startTs := time.Now()
		metricFamilies, err := agent.Metrics(ck.GlobalConfig, ep)
		chkResult.EpMetricsList[i].Took = time.Since(startTs).String()
		if err != nil {
			appendSummary(&chkResult, "Failed to get endpoint metrics from %q: %v", ep, err)
			continue
		}

		metricsMap := make(map[string]MetricOutput)
		for _, metricsName := range metricsNames {

			if mf, ok := metricFamilies[metricsName]; ok {
				output := MetricOutput{
					Type:    mf.GetType().String(),
					Help:    mf.GetHelp(),
					Samples: []MetricSample{},
				}

				for _, m := range mf.GetMetric() {
					sample := MetricSample{
						Labels: map[string]string{},
					}
					for _, l := range m.Label {
						sample.Labels[*l.Name] = *l.Value
					}

					switch mf.GetType() {
					case dto.MetricType_GAUGE:
						sample.Metric = GaugeMetric{
							Value: m.Gauge.GetValue(),
						}
					case dto.MetricType_HISTOGRAM:
						buckets := make([]HistogramBucket, len(m.Histogram.GetBucket()))
						for i, b := range m.Histogram.GetBucket() {
							buckets[i] = HistogramBucket{
								Le:    SafeFloat64(b.GetUpperBound()),
								Count: b.GetCumulativeCount(),
							}
						}

						histogram := HistogramMetric{
							Count:       m.Histogram.GetSampleCount(),
							Sum:         m.Histogram.GetSampleSum(),
							Buckets:     buckets,
							Percentiles: []Percentile{},
						}
						for _, p := range percentileOptions {
							histogram.Percentiles = append(histogram.Percentiles, Percentile{
								Name:  p,
								Value: SafeFloat64(percentileFromHistogram(histogram, float64(p)/100.0)),
							})
						}

						sample.Metric = histogram

					default:
						log.Printf("Metrics name: %s, Unknown metric type: %v", metricsName, mf.GetType())
						continue
					}

					output.Samples = append(output.Samples, sample)
				}

				metricsMap[metricsName] = output
			}
		}
		chkResult.EpMetricsList[i].EpMetrics = metricsMap
	}

	if len(chkResult.Summary) == 0 {
		chkResult.Summary = []string{"Successful"}
	}

	result = chkResult
	return result
}

func percentileFromHistogram(h HistogramMetric, quantile float64) float64 {
	if h.Count == 0 || len(h.Buckets) == 0 {
		return 0
	}

	target := uint64(float64(h.Count) * quantile)
	if target == 0 {
		target = 1
	}

	var prevCount uint64 = 0
	var prevUpper float64 = 0

	for _, b := range h.Buckets {
		if b.Count >= target {
			countInBucket := b.Count - prevCount
			if countInBucket == 0 {
				return float64(b.Le)
			}
			fraction := float64(target-prevCount) / float64(countInBucket)
			return prevUpper + fraction*(float64(b.Le)-prevUpper)
		}
		prevCount = b.Count
		prevUpper = float64(b.Le)
	}

	return float64(h.Buckets[len(h.Buckets)-1].Le)
}

func appendSummary(chkResult *checkResult, format string, v ...any) {
	errMsg := fmt.Sprintf(format, v...)
	log.Println(errMsg)
	chkResult.Summary = append(chkResult.Summary, errMsg)
}
