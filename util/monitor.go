package util

import (
	"runtime"
	"strconv"

	"github.com/mackerelio/go-osstat/memory"
)

type UtilizationReport struct {
	NumCPU       int                     `json:"numCpu"`
	NumGoroutine int                     `json:"numGoroutine"`
	MemorySys    UtilizationReportMemory `json:"memorySys"`
}

type UtilizationReportMemory struct {
	RawData uint64 `json:"rawData"`
	Size    string `json:"size"`
}

func Utilization() (*UtilizationReport, error) {
	numCpu := runtime.NumCPU()
	numGoroutine := runtime.NumGoroutine()
	memory, err := memory.Get()
	if err != nil {
		return nil, err
	}
	memorySys := UtilizationReportMemory{
		RawData: memory.Used / 8,
		Size:    byteUnits(memory.Used/8, 3),
	}

	return &UtilizationReport{
		NumCPU:       numCpu,
		NumGoroutine: numGoroutine,
		MemorySys:    memorySys,
	}, nil
}

func byteUnits(bytes uint64, digits int) string {
	b := float64(bytes)
	if b < (1024 * 0.8) {
		return strconv.FormatFloat(b, 'G', digits, 64) + " B"
	} else if b /= 1024; b < (1024 * 0.8) {
		return strconv.FormatFloat(b, 'G', digits, 64) + " KB"
	} else if b /= 1024; b < (1024 * 0.8) {
		return strconv.FormatFloat(b, 'G', digits, 64) + " MB"
	} else {
		b /= 1024
		return strconv.FormatFloat(b, 'G', digits, 64) + " GB"
	}
}
