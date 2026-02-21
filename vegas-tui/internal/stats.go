package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type statsTickMsg struct{}

type statsDataMsg struct {
	cpuPercent  float64
	memPercent  float64
	memUsed     uint64
	memTotal    uint64
	temp        float64
	diskPercent float64
	diskUsed    uint64
	diskTotal   uint64
	uptime      uint64
	hostname    string
}

type StatsModel struct {
	cpuPercent  float64
	memPercent  float64
	memUsed     uint64
	memTotal    uint64
	temp        float64
	diskPercent float64
	diskUsed    uint64
	diskTotal   uint64
	uptime      uint64
	hostname    string
	loaded      bool
}

func NewStatsModel() StatsModel {
	return StatsModel{}
}

func (m StatsModel) Init() tea.Cmd {
	return tea.Batch(
		fetchStats,
		statsTickCmd(),
	)
}

func statsTickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return statsTickMsg{}
	})
}

func fetchStats() tea.Msg {
	data := statsDataMsg{}

	// CPU
	percents, err := cpu.Percent(0, false)
	if err == nil && len(percents) > 0 {
		data.cpuPercent = percents[0]
	}

	// Memory
	vmem, err := mem.VirtualMemory()
	if err == nil {
		data.memPercent = vmem.UsedPercent
		data.memUsed = vmem.Used
		data.memTotal = vmem.Total
	}

	// Temperature -- gopsutil returns warnings as errors even when data is valid,
	// so check the slice regardless of err.
	temps, _ := host.SensorsTemperatures()
	// Prefer CPU package sensors, fall back to first valid reading.
	cpuSensorKeys := []string{"k10temp_tctl", "coretemp_package_id_0", "cpu_thermal"}
	for _, key := range cpuSensorKeys {
		for _, t := range temps {
			if t.SensorKey == key && t.Temperature > 0 {
				data.temp = t.Temperature
				break
			}
		}
		if data.temp > 0 {
			break
		}
	}
	if data.temp == 0 {
		for _, t := range temps {
			if t.Temperature > 0 {
				data.temp = t.Temperature
				break
			}
		}
	}

	// Disk
	usage, err := disk.Usage("/")
	if err == nil {
		data.diskPercent = usage.UsedPercent
		data.diskUsed = usage.Used
		data.diskTotal = usage.Total
	}

	// Host info
	data.uptime, _ = host.Uptime()
	info, err := host.Info()
	if err == nil {
		data.hostname = info.Hostname
	}

	return data
}

func (m StatsModel) Update(msg tea.Msg) (StatsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			if zone.Get("stats-refresh").InBounds(msg) {
				return m, fetchStats
			}
		}
	case statsTickMsg:
		return m, tea.Batch(fetchStats, statsTickCmd())
	case statsDataMsg:
		m.cpuPercent = msg.cpuPercent
		m.memPercent = msg.memPercent
		m.memUsed = msg.memUsed
		m.memTotal = msg.memTotal
		m.temp = msg.temp
		m.diskPercent = msg.diskPercent
		m.diskUsed = msg.diskUsed
		m.diskTotal = msg.diskTotal
		m.uptime = msg.uptime
		m.hostname = msg.hostname
		m.loaded = true
	}
	return m, nil
}

func (m StatsModel) View(width, height int) string {
	if !m.loaded {
		return theme.DimStyle.Render("  Scanning systems...")
	}

	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" S.Y.S.T.E.M.  S.T.A.T.U.S. "))
	b.WriteString("\n\n")

	barWidth := width - 30
	if barWidth < 20 {
		barWidth = 20
	}
	if barWidth > 40 {
		barWidth = 40
	}

	// CPU
	b.WriteString(renderGauge("PROCESSOR", m.cpuPercent, barWidth))
	b.WriteString("\n\n")

	// Memory
	b.WriteString(renderGauge("MEMORY", m.memPercent, barWidth))
	memDetail := fmt.Sprintf("  %s / %s",
		formatBytes(m.memUsed),
		formatBytes(m.memTotal),
	)
	b.WriteString("\n")
	b.WriteString(theme.DimStyle.Render("               " + memDetail))
	b.WriteString("\n\n")

	// Temperature
	tempPct := m.temp / 100.0 * 100
	if tempPct > 100 {
		tempPct = 100
	}
	b.WriteString(renderTempGauge("TEMPERATURE", m.temp, tempPct, barWidth))
	b.WriteString("\n\n")

	// Disk
	b.WriteString(renderGauge("STORAGE", m.diskPercent, barWidth))
	diskDetail := fmt.Sprintf("  %s / %s",
		formatBytes(m.diskUsed),
		formatBytes(m.diskTotal),
	)
	b.WriteString("\n")
	b.WriteString(theme.DimStyle.Render("               " + diskDetail))
	b.WriteString("\n\n")

	// Divider
	b.WriteString(theme.DimStyle.Render(" " + strings.Repeat("─", barWidth+20)))
	b.WriteString("\n\n")

	// System info
	b.WriteString(fmt.Sprintf(" %s  %s\n",
		theme.GaugeLabel.Render("HOSTNAME:"),
		theme.BaseStyle.Render(m.hostname),
	))
	b.WriteString(fmt.Sprintf(" %s  %s\n",
		theme.GaugeLabel.Render("UPTIME:"),
		theme.BaseStyle.Render(formatUptime(m.uptime)),
	))

	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(zone.Mark("stats-refresh", theme.BaseStyle.Render("[ REFRESH ]")))

	return b.String()
}

func renderGauge(label string, percent float64, barWidth int) string {
	filled := int(percent / 100.0 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}

	bar := theme.GaugeFilled.Render(strings.Repeat("█", filled)) +
		theme.GaugeEmpty.Render(strings.Repeat("░", barWidth-filled))

	pctStr := fmt.Sprintf("%5.1f%%", percent)
	style := theme.BaseStyle
	if percent > 90 {
		style = theme.GaugeCritical
	} else if percent > 75 {
		style = theme.GaugeWarning
	}

	return fmt.Sprintf(" %s  %s %s",
		theme.GaugeLabel.Render(label),
		bar,
		style.Render(pctStr),
	)
}

func renderTempGauge(label string, temp, percent float64, barWidth int) string {
	filled := int(percent / 100.0 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}

	bar := theme.GaugeFilled.Render(strings.Repeat("█", filled)) +
		theme.GaugeEmpty.Render(strings.Repeat("░", barWidth-filled))

	tempStr := fmt.Sprintf("%5.1f°C", temp)
	style := theme.BaseStyle
	if temp > 80 {
		style = theme.GaugeCritical
	} else if temp > 60 {
		style = theme.GaugeWarning
	}

	return fmt.Sprintf(" %s  %s %s",
		theme.GaugeLabel.Render(label),
		bar,
		style.Render(tempStr),
	)
}

func formatBytes(b uint64) string {
	const gb = 1024 * 1024 * 1024
	const mb = 1024 * 1024
	if b >= gb {
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	}
	return fmt.Sprintf("%.0f MB", float64(b)/float64(mb))
}

func formatUptime(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, secs)
	}
	return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
}
