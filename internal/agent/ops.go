package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"runtime"

	"github.com/dursuntokgoz/OpenControl/internal/providers"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
)

// Whitelisted operation names.
const (
	OpPing          = "ping"
	OpSysInfo       = "sysinfo"
	OpServiceList   = "service.list"
	OpServiceAction = "service.action"
)

// RegisterBuiltinOps registers the standard read-only operations.
func RegisterBuiltinOps(s *Server, svc providers.ServiceControl) {
	s.Register(OpPing, handlePing)
	s.Register(OpSysInfo, handleSysInfo)
	s.Register(OpServiceList, func(ctx context.Context, params json.RawMessage) (any, error) {
		services, err := svc.List(ctx)
		if err != nil {
			return nil, err
		}
		return ServiceListResult{Services: services}, nil
	})
	s.Register(OpServiceAction, func(ctx context.Context, params json.RawMessage) (any, error) {
		var p ServiceActionParams
		if len(params) > 0 {
			if err := json.Unmarshal(params, &p); err != nil {
				return nil, err
			}
		}
		unit := providers.ServiceUnit(p.Unit)
		action := providers.ServiceAction(p.Action)
		if err := svc.Apply(ctx, providers.ServiceParams{Unit: unit, Action: action}); err != nil {
			return nil, err
		}
		return map[string]string{"status": "ok"}, nil
	})
}

func handlePing(_ context.Context, _ json.RawMessage) (any, error) {
	hostName, _ := os.Hostname()
	return PingResult{Pong: true, Host: hostName}, nil
}

func handleSysInfo(ctx context.Context, params json.RawMessage) (any, error) {
	var p SysInfoParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
	}
	res := SysInfoResult{OS: runtime.GOOS}

	if hi, err := host.InfoWithContext(ctx); err == nil {
		res.Hostname = hi.Hostname
		res.Platform = hi.Platform + " " + hi.PlatformVersion
		res.KernelVersion = hi.KernelVersion
		res.UptimeSec = hi.Uptime
	}
	if hostName, err := os.Hostname(); err == nil && res.Hostname == "" {
		res.Hostname = hostName
	}
	res.CPUCount = runtime.NumCPU()
	if pct, err := cpu.PercentWithContext(ctx, 0, false); err == nil && len(pct) > 0 {
		res.CPUPercent = pct[0]
	}
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		res.MemTotalMB = vm.Total / (1 << 20)
		res.MemUsedMB = (vm.Total - vm.Available) / (1 << 20)
	}
	if avg, err := load.AvgWithContext(ctx); err == nil {
		res.Load1 = avg.Load1
	}
	if du, err := disk.UsageWithContext(ctx, "/"); err == nil {
		res.DiskRootPct = du.UsedPercent
	}
	filterSections(&res, p.Include)
	slog.Debug("sysinfo collected")
	return res, nil
}

// filterSections zeroes out sections the caller did not request.
func filterSections(res *SysInfoResult, include []string) {
	if len(include) == 0 {
		return
	}
	want := make(map[string]bool, len(include))
	for _, s := range include {
		want[s] = true
	}
	if !want["cpu"] {
		res.CPUCount = 0
		res.CPUPercent = 0
	}
	if !want["mem"] {
		res.MemTotalMB = 0
		res.MemUsedMB = 0
	}
	if !want["load"] {
		res.Load1 = 0
	}
	if !want["disk"] {
		res.DiskRootPct = 0
	}
	if !want["host"] {
		res.Hostname = ""
		res.Platform = ""
		res.KernelVersion = ""
		res.UptimeSec = 0
	}
}
