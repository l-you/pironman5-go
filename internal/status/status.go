package status

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	gocpu "github.com/shirou/gopsutil/v4/cpu"
	godisk "github.com/shirou/gopsutil/v4/disk"
	gomem "github.com/shirou/gopsutil/v4/mem"
	gonet "github.com/shirou/gopsutil/v4/net"
)

type Snapshot struct {
	CPUPercent    float64
	CPUTempC      float64
	MemoryTotal   uint64
	MemoryUsed    uint64
	MemoryPercent float64
	Disks         []Disk
	IPs           []IP
	FanRPM        int
}

type Disk struct {
	Name    string
	Mount   string
	Total   uint64
	Used    uint64
	Percent float64
}

type IP struct {
	Interface string
	Address   string
}

type Sampler interface {
	Snapshot(context.Context) (Snapshot, error)
	CPUTemperatureC(context.Context) (float64, error)
}

type SystemSampler struct{}

func (SystemSampler) Snapshot(ctx context.Context) (Snapshot, error) {
	cpuPercent, err := cpuPercent(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	memory, err := gomem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read memory status: %w", err)
	}
	temp, _ := SystemSampler{}.CPUTemperatureC(ctx)
	disks := readDisks(ctx)
	ips := readIPs(ctx)
	return Snapshot{
		CPUPercent:    cpuPercent,
		CPUTempC:      temp,
		MemoryTotal:   memory.Total,
		MemoryUsed:    memory.Used,
		MemoryPercent: memory.UsedPercent,
		Disks:         disks,
		IPs:           ips,
	}, nil
}

func (SystemSampler) CPUTemperatureC(context.Context) (float64, error) {
	data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return 0, fmt.Errorf("read cpu temperature: %w", err)
	}
	milli, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return 0, fmt.Errorf("parse cpu temperature: %w", err)
	}
	return milli / 1000, nil
}

func cpuPercent(ctx context.Context) (float64, error) {
	values, err := gocpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return 0, fmt.Errorf("read cpu usage: %w", err)
	}
	if len(values) == 0 {
		return 0, nil
	}
	return values[0], nil
}

func readDisks(ctx context.Context) []Disk {
	partitions, err := godisk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil
	}
	out := make([]Disk, 0, len(partitions))
	seen := make(map[string]struct{})
	for _, partition := range partitions {
		if _, ok := seen[partition.Mountpoint]; ok {
			continue
		}
		seen[partition.Mountpoint] = struct{}{}
		usage, err := godisk.UsageWithContext(ctx, partition.Mountpoint)
		if err != nil {
			continue
		}
		out = append(out, Disk{
			Name:    filepath.Base(partition.Device),
			Mount:   partition.Mountpoint,
			Total:   usage.Total,
			Used:    usage.Used,
			Percent: usage.UsedPercent,
		})
	}
	return out
}

func readIPs(ctx context.Context) []IP {
	interfaces, err := gonet.InterfacesWithContext(ctx)
	if err != nil {
		return nil
	}
	out := make([]IP, 0, len(interfaces))
	for _, iface := range interfaces {
		if iface.Name == "lo" {
			continue
		}
		for _, addr := range iface.Addrs {
			host := addr.Addr
			if parsed, _, err := net.ParseCIDR(addr.Addr); err == nil {
				host = parsed.String()
			}
			if net.ParseIP(host).To4() == nil {
				continue
			}
			out = append(out, IP{Interface: iface.Name, Address: host})
		}
	}
	return out
}
