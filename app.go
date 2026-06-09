package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
)

// App struct
type App struct {
	ctx          context.Context
	lastCPUIdle  uint64
	lastCPUTotal uint64
	mu           sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetSystemStats fetches real-time CPU, RAM, and Uptime on Linux
func (a *App) GetSystemStats() (map[string]interface{}, error) {
	// Read MemInfo
	memTotal, memAvail, err := readMemInfo()
	memPercent := 0.0
	if err == nil && memTotal > 0 {
		memPercent = float64(memTotal-memAvail) / float64(memTotal) * 100.0
	}

	// Read CPU load
	cpuPercent, err := a.calculateCPULoad()
	if err != nil {
		cpuPercent = 0.0
	}

	// Get Uptime
	uptimeStr := "Unknown"
	if uptimeData, err := os.ReadFile("/proc/uptime"); err == nil {
		var uptimeSecs float64
		if _, err = fmt.Sscanf(string(uptimeData), "%f", &uptimeSecs); err == nil {
			uptimeStr = formatUptime(uptimeSecs)
		}
	}

	return map[string]interface{}{
		"cpu":    cpuPercent,
		"memory": memPercent,
		"uptime": uptimeStr,
	}, nil
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

func (a *App) calculateCPULoad() (float64, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	idle2, total2, err := readCPUStats()
	if err != nil {
		return 0, err
	}

	idle1, total1 := a.lastCPUIdle, a.lastCPUTotal
	a.lastCPUIdle, a.lastCPUTotal = idle2, total2

	if total1 == 0 || total2 == total1 {
		return 0, nil
	}

	totalDiff := float64(total2 - total1)
	idleDiff := float64(idle2 - idle1)

	return (totalDiff - idleDiff) / totalDiff * 100.0, nil
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

