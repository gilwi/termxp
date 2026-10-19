package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	collector  MetricsCollector
	allowClose atomic.Bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	var collector MetricsCollector
	if runtime.GOOS == "linux" {
		collector = NewLinuxMetricsCollector()
	} else {
		collector = NewFallbackMetricsCollector()
	}
	return &App{
		collector: collector,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetSystemStats fetches real-time CPU, RAM, and Uptime for the current process
func (a *App) GetSystemStats() (SystemStats, error) {
	return a.collector.GetStats()
}

// SystemStats describes only the Go application process, excluding shells/webview.
type SystemStats struct {
	CPU       *float64 `json:"cpu"`
	Memory    *float64 `json:"memory"`
	MemoryRaw string   `json:"memoryRaw"`
	Uptime    string   `json:"uptime"`
}

func (a *App) beforeClose(ctx context.Context) bool {
	if a.allowClose.Load() {
		return false
	}
	wailsruntime.EventsEmit(ctx, "app:closing")
	return true
}
func (a *App) ConfirmClose() { a.allowClose.Store(true); wailsruntime.Quit(a.ctx) }

// MetricsCollector defines the interface for OS stats retrieval
type MetricsCollector interface {
	GetStats() (SystemStats, error)
}

// LinuxMetricsCollector implements MetricsCollector for Linux systems via /proc
type LinuxMetricsCollector struct {
	startTime    time.Time
	lastCPUTotal uint64
	lastProcTime uint64
	mu           sync.Mutex
}

// NewLinuxMetricsCollector creates a Linux stat collector
func NewLinuxMetricsCollector() *LinuxMetricsCollector {
	return &LinuxMetricsCollector{
		startTime: time.Now(),
	}
}

// GetStats retrieves stats on Linux
func (l *LinuxMetricsCollector) GetStats() (SystemStats, error) {
	stats := SystemStats{MemoryRaw: "Unavailable", Uptime: formatUptime(time.Since(l.startTime).Seconds())}
	if rss, err := readProcessMemory(); err == nil {
		stats.MemoryRaw = formatMemory(rss)
		if total, _, err := readMemInfo(); err == nil && total > 0 {
			percent := float64(rss) / float64(total) * 100
			stats.Memory = &percent
		}
	}
	if cpu, err := l.calculateProcessCPULoad(); err == nil {
		stats.CPU = &cpu
	}
	return stats, nil
}

func (l *LinuxMetricsCollector) calculateProcessCPULoad() (float64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	proc2, total2, err := readProcessCPUStats()
	if err != nil {
		return 0, err
	}

	proc1, total1 := l.lastProcTime, l.lastCPUTotal
	l.lastProcTime, l.lastCPUTotal = proc2, total2

	if total1 == 0 || total2 <= total1 {
		return 0, nil
	}

	totalDiff := float64(total2 - total1)
	procDiff := float64(proc2 - proc1)

	return (procDiff / totalDiff) * 100.0 * float64(runtime.NumCPU()), nil
}

// FallbackMetricsCollector implements MetricsCollector for macOS/Windows
type FallbackMetricsCollector struct {
	startTime time.Time
}

// NewFallbackMetricsCollector creates a fallback stat collector
func NewFallbackMetricsCollector() *FallbackMetricsCollector {
	return &FallbackMetricsCollector{
		startTime: time.Now(),
	}
}

// GetStats retrieves stats for fallback systems
func (f *FallbackMetricsCollector) GetStats() (SystemStats, error) {
	return SystemStats{MemoryRaw: "Unavailable", Uptime: formatUptime(time.Since(f.startTime).Seconds())}, nil
}

// Helper formatting and reading functions

func formatMemory(kb uint64) string {
	if kb < 1024*1024 {
		return fmt.Sprintf("%d MB", kb/1024)
	}
	return fmt.Sprintf("%.1f GB", float64(kb)/(1024*1024))
}

func readProcessMemory() (rss uint64, err error) {
	file, err := os.Open("/proc/self/status")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VmRSS:") {
			_, _ = fmt.Sscanf(line, "VmRSS: %d", &rss)
			return rss, nil
		}
	}
	return 0, fmt.Errorf("VmRSS not found")
}

func readProcessCPUStats() (procTime uint64, totalTime uint64, err error) {
	_, totalTime, err = readCPUStats()
	if err != nil {
		return 0, 0, err
	}

	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0, 0, err
	}
	return parseProcessCPUStats(string(data), totalTime)
}

func parseProcessCPUStats(content string, totalTime uint64) (uint64, uint64, error) {
	lastParen := strings.LastIndex(content, ")")
	if lastParen == -1 || lastParen+1 >= len(content) {
		return 0, 0, fmt.Errorf("invalid proc self stat format: missing closing paren")
	}

	fields := strings.Fields(content[lastParen+1:])
	if len(fields) < 13 {
		return 0, 0, fmt.Errorf("invalid proc self stat format: not enough fields")
	}

	utime, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	stime, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	return utime + stime, totalTime, nil
}

func readMemInfo() (total, avail uint64, err error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	var foundTotal, foundAvail bool
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			_, _ = fmt.Sscanf(line, "MemTotal: %d", &total)
			foundTotal = true
		} else if strings.HasPrefix(line, "MemAvailable:") {
			_, _ = fmt.Sscanf(line, "MemAvailable: %d", &avail)
			foundAvail = true
		}
		if foundTotal && foundAvail {
			break
		}
	}
	return total, avail, scanner.Err()
}

func readCPUStats() (idle, total uint64, err error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 9 && fields[0] == "cpu" {
			counters := make([]uint64, 8)
			for i := range counters {
				value, err := strconv.ParseUint(fields[i+1], 10, 64)
				if err != nil {
					return 0, 0, fmt.Errorf("invalid CPU counter: %w", err)
				}
				counters[i] = value
				total += value
			}
			idle = counters[3] + counters[4]
			return idle, total, nil
		}
	}
	return 0, 0, fmt.Errorf("invalid proc stat format")
}

func formatUptime(secs float64) string {
	days := int(secs) / (24 * 3600)
	hours := (int(secs) % (24 * 3600)) / 3600
	mins := (int(secs) % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
