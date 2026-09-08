package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

const (
	StartRPS      = 1900
	StepRPS       = 100
	MaxRPS        = 10000
	StepDuration  = 10 * time.Second
	MaxLatencyMs  = 100.0
	MaxErrorPct   = 1.0
	CooldownPause = 20 * time.Second
)

func main() {
	targets := []string{"targets-checkin.json", "targets-report-success.json"}

	for _, path := range targets {
		rps, stats, err := rampUp(path)
		if err != nil {
			log.Printf("%s: %v", path, err)
		}
		logSummary(path, rps, stats)
	}
}

func rampUp(path string) (int, vegeta.Metrics, error) {
	if _, err := os.Stat(path); err != nil {
		return 0, vegeta.Metrics{}, err
	}

	lastStable := 0
	var lastMetrics vegeta.Metrics

	for rps := StartRPS; rps <= MaxRPS; rps += StepRPS {
		log.Printf("%s: raising RPS to %d", path, rps)

		metrics, err := runStep(path, rps)
		if err != nil {
			return lastStable, lastMetrics, err
		}

		logStep(path, rps, metrics)

		if failed, reason := exceedsBudget(metrics); failed {
			log.Printf("%s: %s at %d rps, stopping ramp-up", path, reason, rps)
			break
		}

		lastStable = rps
		lastMetrics = metrics
		time.Sleep(CooldownPause)
	}

	time.Sleep(CooldownPause)

	return lastStable, lastMetrics, nil
}

func runStep(path string, rps int) (vegeta.Metrics, error) {
	file, err := os.Open(path)
	if err != nil {
		return vegeta.Metrics{}, err
	}
	defer file.Close()

	var metrics vegeta.Metrics
	attacker := vegeta.NewAttacker()
	defer attacker.Stop()

	pacer := vegeta.ConstantPacer{Freq: rps, Per: time.Second}
	for res := range attacker.Attack(vegeta.NewJSONTargeter(file, nil, nil), pacer, StepDuration, "ramp-up") {
		metrics.Add(res)
	}
	metrics.Close()

	return metrics, nil
}

func logStep(path string, rps int, m vegeta.Metrics) {
	// errorPct := (1.0 - m.Success) * 100.0
	// log.Printf("%s: step rps=%d requests=%d success=%.2f%% errors=%.2f%% status=%v rate=%.1f throughput=%.1f duration=%s wait=%s",
	// 	path, rps, m.Requests, m.Success*100.0, errorPct, m.StatusCodes, m.Rate, m.Throughput, m.Duration, m.Wait)
	// log.Printf("%s: step rps=%d latencies mean=%s p50=%s p90=%s p95=%s p99=%s max=%s min=%s bytes_in=%.1f bytes_out=%.1f",
	// 	path, rps, m.Latencies.Mean, m.Latencies.P50, m.Latencies.P90, m.Latencies.P95, m.Latencies.P99, m.Latencies.Max, m.Latencies.Min, m.BytesIn.Mean, m.BytesOut.Mean)
	if len(m.Errors) > 0 {
		log.Printf("%s: step rps=%d error_samples=%q", path, rps, m.Errors)
	}
}

func logSummary(path string, rps int, m vegeta.Metrics) {
	if rps == 0 {
		log.Printf("%s: no stable RPS, budget exceeded already at %d rps", path, StartRPS)
		return
	}
	errorPct := (1.0 - m.Success) * 100.0
	log.Printf("%s: stable %d rps requests=%d success=%.2f%% errors=%.2f%% status=%v rate=%.1f throughput=%.1f duration=%s wait=%s",
		path, rps, m.Requests, m.Success*100.0, errorPct, m.StatusCodes, m.Rate, m.Throughput, m.Duration, m.Wait)
	log.Printf("%s: stable %d rps latencies mean=%s p50=%s p90=%s p95=%s p99=%s max=%s min=%s bytes_in=%.1f bytes_out=%.1f",
		path, rps, m.Latencies.Mean, m.Latencies.P50, m.Latencies.P90, m.Latencies.P95, m.Latencies.P99, m.Latencies.Max, m.Latencies.Min, m.BytesIn.Mean, m.BytesOut.Mean)
	if len(m.Errors) > 0 {
		log.Printf("%s: stable %d rps error_samples=%q", path, rps, m.Errors)
	}
}

func exceedsBudget(m vegeta.Metrics) (bool, string) {
	latencyMs := float64(m.Latencies.Mean.Microseconds()) / 1000.0
	errorPct := (1.0 - m.Success) * 100.0

	reasons := make([]string, 0, 2)
	if latencyMs > MaxLatencyMs {
		reasons = append(reasons, fmt.Sprintf("mean latency %.2fms > %.2fms", latencyMs, MaxLatencyMs))
	}
	if errorPct > MaxErrorPct {
		reasons = append(reasons, fmt.Sprintf("errors %.2f%% > %.2f%%", errorPct, MaxErrorPct))
	}

	if len(reasons) == 0 {
		return false, ""
	}
	return true, strings.Join(reasons, "; ")
}
