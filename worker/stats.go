package worker

import (
	"Mine-Cube/logger"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

var statsLog = logger.GetLogger("worker.stats")

type Stats struct {
	MemStats  *mem.VirtualMemoryStat
	DiskStats *disk.UsageStat
	CpuStats  []cpu.TimesStat
	LoadStats *load.AvgStat
	TaskCount int
}

func (s *Stats) MemTotalKb() uint64 {
	return s.MemStats.Total / 1024
}

func (s *Stats) MemAvailableKb() uint64 {
	return s.MemStats.Available / 1024
}

func (s *Stats) MemUsedKb() uint64 {
	return s.MemStats.Used / 1024
}

func (s *Stats) MemUsedPercent() uint64 {
	return uint64(s.MemStats.UsedPercent)
}

func (s *Stats) DiskTotal() uint64 {
	return s.DiskStats.Total
}

func (s *Stats) DiskFree() uint64 {
	return s.DiskStats.Free
}

func (s *Stats) DiskUsed() uint64 {
	return s.DiskStats.Used
}

// NOTE: this problem was solved in “Accurate Calculation of CPU Usage Given in Percentage in Linux” post by (http://mng.bz/xj17)
// 1 Sum the values for the idle states.
// 2 Sum the values for the non-idle states.
// 3 Sum the total of idle and non-idle states.
// 4 Subtract the idle from the total and divide the result by the total.
func (s *Stats) CpuUsage() float64 {
	if len(s.CpuStats) == 0 {
		return 0.00
	}

	// Get the first CPU stats (combined)
	cpuStat := s.CpuStats[0]

	idle := cpuStat.Idle + cpuStat.Iowait
	nonIdle := cpuStat.User + cpuStat.Nice + cpuStat.System +
		cpuStat.Irq + cpuStat.Softirq + cpuStat.Steal
	total := idle + nonIdle

	if total == 0 {
		return 0.00
	}
	return (total - idle) / total * 100
}

func GetMemoryInfo() *mem.VirtualMemoryStat {
	memstats, err := mem.VirtualMemory()

	if err != nil {
		statsLog.Errorf("Error reading memory info: %v", err)
		return &mem.VirtualMemoryStat{}
	}

	return memstats
}

func GetDiskInfo() *disk.UsageStat {
	diskstats, err := disk.Usage("/")

	if err != nil {
		statsLog.Errorf("Error reading disk info: %v", err)
		return &disk.UsageStat{}
	}

	return diskstats
}

func GetCpuStats() []cpu.TimesStat {
	stats, err := cpu.Times(false)

	if err != nil {
		statsLog.Errorf("Error reading CPU stats: %v", err)
		return []cpu.TimesStat{}
	}

	return stats
}

func GetLoadAvg() *load.AvgStat {
	loadavg, err := load.Avg()

	if err != nil {
		statsLog.Errorf("Error reading load average: %v", err)
		return &load.AvgStat{}
	}

	return loadavg
}

func GetStats() *Stats {
	return &Stats{
		MemStats:  GetMemoryInfo(),
		DiskStats: GetDiskInfo(),
		CpuStats:  GetCpuStats(),
		LoadStats: GetLoadAvg(),
	}
}
