package providers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	systemctlBinary  = "/usr/bin/systemctl"
	serviceCmdTimeut = 30 * time.Second
	maxServiceOutput = 64 * 1024
)

var errUnitNotAllowed = errors.New("service unit is not allowlisted")

type limitedBuffer struct {
	buf      bytes.Buffer
	max      int
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.max - b.buf.Len()
	if len(p) > remaining {
		p = p[:remaining]
		b.exceeded = true
	}
	_, err := b.buf.Write(p)
	return n, err
}

func (b *limitedBuffer) String() string { return b.buf.String() }

func (b *limitedBuffer) overflowed() bool { return b.exceeded }

type ServiceUnit string

type ServiceAction string

const (
	ServiceStart   ServiceAction = "start"
	ServiceStop    ServiceAction = "stop"
	ServiceRestart ServiceAction = "restart"
	ServiceEnable  ServiceAction = "enable"
)

type ServiceParams struct {
	Unit   ServiceUnit   `json:"unit"`
	Action ServiceAction `json:"action"`
}

type ServiceStatus struct {
	Unit          ServiceUnit  `json:"unit"`
	LoadState     string       `json:"loadState"`
	ActiveState   ServiceState `json:"activeState"`
	SubState      string       `json:"subState"`
	UnitFileState string       `json:"unitFileState"`
}

func ServiceUnits() []ServiceUnit {
	return []ServiceUnit{
		"nginx.service", "apache2.service", "httpd.service", "mariadb.service",
		"postgresql.service", "postfix.service", "dovecot.service", "bind9.service", "named.service",
	}
}

func (u ServiceUnit) Validate() error {
	for _, allowed := range ServiceUnits() {
		if u == allowed {
			return nil
		}
	}
	return errUnitNotAllowed
}

func (p ServiceParams) Validate() error {
	if err := p.Unit.Validate(); err != nil {
		return err
	}
	switch p.Action {
	case ServiceStart, ServiceStop, ServiceRestart, ServiceEnable:
		return nil
	default:
		return fmt.Errorf("service action is not allowlisted")
	}
}

// SystemdServiceControl implements ServiceControl via systemctl.
// It uses argument arrays only (no shell) and honors strict allowlists.
type SystemdServiceControl struct {
	// SystemctlPath allows tests to point at a stub; production uses the default.
	SystemctlPath string
}

func (s *SystemdServiceControl) systemctl() string {
	if s.SystemctlPath != "" {
		return s.SystemctlPath
	}
	return systemctlBinary
}

func (s *SystemdServiceControl) run(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, serviceCmdTimeut)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.systemctl(), args...)
	buf := &limitedBuffer{max: maxServiceOutput}
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(buf.String())
		if len(msg) > maxServiceOutput {
			msg = msg[:maxServiceOutput]
		}
		if msg != "" {
			return msg, fmt.Errorf("systemctl %s: %w: %s", strings.Join(args, " "), err, msg)
		}
		return "", fmt.Errorf("systemctl %s: %w", strings.Join(args, " "), err)
	}
	if buf.overflowed() {
		return "", fmt.Errorf("systemctl %s: output exceeded %d bytes", strings.Join(args, " "), maxServiceOutput)
	}
	return strings.TrimSpace(buf.String()), nil
}

// List returns statuses for every allowlisted unit present on the host.
func (s *SystemdServiceControl) List(ctx context.Context) ([]ServiceStatus, error) {
	units := ServiceUnits()
	args := []string{"show", "--property=Id,LoadState,ActiveState,SubState,UnitFileState", "--"}
	args = append(args, unitStrings(units)...)
	out, err := s.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	blocks := strings.Split(out, "\n\n")
	statuses := make([]ServiceStatus, 0, len(units))
	for i, block := range blocks {
		if i >= len(units) {
			break
		}
		statuses = append(statuses, parseShowBlock(units[i], block))
	}
	return statuses, nil
}

// Status returns the state of one allowlisted unit.
func (s *SystemdServiceControl) Status(ctx context.Context, unit ServiceUnit) (ServiceStatus, error) {
	if err := unit.Validate(); err != nil {
		return ServiceStatus{}, err
	}
	out, err := s.run(ctx, "show", "--property=Id,LoadState,ActiveState,SubState,UnitFileState", string(unit))
	if err != nil {
		return ServiceStatus{}, err
	}
	return parseShowBlock(unit, out), nil
}

// Apply performs an allowlisted action on an allowlisted unit.
func (s *SystemdServiceControl) Apply(ctx context.Context, params ServiceParams) error {
	if err := params.Validate(); err != nil {
		return err
	}
	_, err := s.run(ctx, string(params.Action), string(params.Unit))
	return err
}

func unitStrings(units []ServiceUnit) []string {
	out := make([]string, len(units))
	for i, u := range units {
		out[i] = string(u)
	}
	return out
}

func parseShowBlock(unit ServiceUnit, block string) ServiceStatus {
	st := ServiceStatus{Unit: unit, ActiveState: ServiceUnknown}
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "Id":
			if value != "" {
				st.Unit = ServiceUnit(value)
			}
		case "LoadState":
			st.LoadState = value
		case "ActiveState":
			st.ActiveState = ServiceState(value)
		case "SubState":
			st.SubState = value
		case "UnitFileState":
			st.UnitFileState = value
		}
	}
	if st.ActiveState == "" {
		st.ActiveState = ServiceUnknown
	}
	return st
}
