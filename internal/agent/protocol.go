// Package agent implements the JSON protocol spoken between panel-api and the
// privileged panel-agent over a Unix domain socket. Every operation must be
// registered in the server's whitelist before the agent will execute it.
package agent

import "encoding/json"

// Request is a single operation request.
type Request struct {
	// ID correlates responses with requests.
	ID uint64 `json:"id"`
	// Op names a whitelisted operation.
	Op string `json:"op"`
	// Token authenticates the caller (shared secret).
	Token string `json:"token,omitempty"`
	// Params carries the typed parameters marshalled by the caller.
	Params json.RawMessage `json:"params,omitempty"`
}

// Response is the reply to a Request.
type Response struct {
	ID     uint64          `json:"id"`
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// PingParams/PingResult implement the trivial liveness operation.
type PingParams struct{}

// PingResult confirms agent identity.
type PingResult struct {
	Pong bool   `json:"pong"`
	Host string `json:"host"`
}

// SysInfoParams selects which sections to include (empty = all).
type SysInfoParams struct {
	Include []string `json:"include,omitempty"`
}

// SysInfoResult aggregates host vitals for dashboards.
type SysInfoResult struct {
	Hostname      string  `json:"hostname"`
	OS            string  `json:"os"`
	Platform      string  `json:"platform"`
	KernelVersion string  `json:"kernelVersion"`
	CPUCount      int     `json:"cpuCount"`
	CPUPercent    float64 `json:"cpuPercent"`
	MemTotalMB    uint64  `json:"memTotalMb"`
	MemUsedMB     uint64  `json:"memUsedMb"`
	Load1         float64 `json:"load1"`
	UptimeSec     uint64  `json:"uptimeSec"`
	DiskRootPct   float64 `json:"diskRootPct"`
}
