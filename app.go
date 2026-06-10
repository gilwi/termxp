package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// App struct
type App struct {
	ctx          context.Context
	lastCPUTotal uint64
	lastProcTime uint64
	startTime    time.Time
	mu           sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		startTime: time.Now(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetSystemStats fetches real-time CPU, RAM, and Uptime for the current process
func (a *App) GetSystemStats() (map[string]interface{}, error) {
	// Read Process RAM (RSS)
	memRSS, err := readProcessMemory()
	if err != nil {
		memRSS = 0
	}
	// Convert RSS (KB) to a percentage of total system memory for the bar,
	// or just return the MB value? The UI expects a percentage for the bar.
	// Let's get total memory to calculate percentage.
	memTotal, _, _ := readMemInfo()
	memPercent := 0.0
	if memTotal > 0 {
		memPercent = (float64(memRSS) / float64(memTotal)) * 100.0
	}

	// Read Process CPU load
	cpuPercent, err := a.calculateProcessCPULoad()
	if err != nil {
		cpuPercent = 0.0
	}

	// Get Process Uptime
	uptimeStr := formatUptime(time.Since(a.startTime).Seconds())

	return map[string]interface{}{
		"cpu":       cpuPercent,
		"memory":    memPercent,
		"memoryRaw": formatMemory(memRSS),
		"uptime":    uptimeStr,
	}, nil
}

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
	// Get total system time
	_, totalTime, err = readCPUStats()
	if err != nil {
		return 0, 0, err
	}

	// Get process time (utime + stime)
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 15 {
		return 0, 0, fmt.Errorf("invalid proc self stat format")
	}

	var utime, stime uint64
	_, _ = fmt.Sscanf(fields[13], "%d", &utime)
	_, _ = fmt.Sscanf(fields[14], "%d", &stime)

	return utime + stime, totalTime, nil
}

func (a *App) calculateProcessCPULoad() (float64, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	proc2, total2, err := readProcessCPUStats()
	if err != nil {
		return 0, err
	}

	proc1, total1 := a.lastProcTime, a.lastCPUTotal
	a.lastProcTime, a.lastCPUTotal = proc2, total2

	if total1 == 0 || total2 <= total1 {
		return 0, nil
	}

	totalDiff := float64(total2 - total1)
	procDiff := float64(proc2 - proc1)

	// CPU percentage relative to total system capacity
	return (procDiff / totalDiff) * 100.0, nil
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
		if len(fields) > 4 && fields[0] == "cpu" {
			var user, nice, system, idleVal, iowait, irq, softirq, steal uint64
			_, _ = fmt.Sscanf(fields[1], "%d", &user)
			_, _ = fmt.Sscanf(fields[2], "%d", &nice)
			_, _ = fmt.Sscanf(fields[3], "%d", &system)
			_, _ = fmt.Sscanf(fields[4], "%d", &idleVal)
			_, _ = fmt.Sscanf(fields[5], "%d", &iowait)
			_, _ = fmt.Sscanf(fields[6], "%d", &irq)
			_, _ = fmt.Sscanf(fields[7], "%d", &softirq)
			_, _ = fmt.Sscanf(fields[8], "%d", &steal)

			idle = idleVal + iowait
			total = idle + user + nice + system + irq + softirq + steal
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
