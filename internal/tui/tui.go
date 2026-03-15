package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mopeyjellyfish/hookr/internal/call"
	"github.com/mopeyjellyfish/hookr/internal/contract"
)

type Config struct {
	SchemaPath        string
	PluginPath        string
	HostFixturePath   string
	Hash              string
	AllowUnsigned     bool
	FlatcPath         string
	IncludePaths      []string
	Package           string
	PluginService     string
	OptionalAttribute string
	Stdin             io.Reader
	Stdout            io.Writer
}

type methodItem struct {
	method contract.Method
}

func (i methodItem) Title() string {
	required := "required"
	if i.method.Optional {
		required = "optional"
	}
	return fmt.Sprintf("%s [%s]", i.method.Name, required)
}

func (i methodItem) Description() string {
	return fmt.Sprintf("id=%d  %s -> %s", i.method.ID, i.method.RequestType, i.method.ResponseType)
}

func (i methodItem) FilterValue() string {
	return i.method.Name
}

type callResultMsg struct {
	result call.Result
	err    error
}

type editorResultMsg struct {
	content string
	err     error
}

type reloadResultMsg struct {
	session *call.Session
	modTime time.Time
	err     error
}

type fileCheckMsg struct {
	modTime time.Time
	err     error
}

type loopUpdateMsg struct {
	stats loopStats
	err   error
	done  bool
}

type loopStats struct {
	Iterations         int
	Errors             int
	Started            time.Time
	Elapsed            time.Duration
	Last               time.Duration
	Min                time.Duration
	Max                time.Duration
	Total              time.Duration
	RequestBytes       int
	ResponseBytes      int
	TotalRequestBytes  int64
	TotalResponseBytes int64
}

type loopController struct {
	stop    chan struct{}
	updates chan loopUpdateMsg
}

type paneFocus int

const (
	focusMethods paneFocus = iota
	focusRequest
	focusResponse
	focusDebug
)

type model struct {
	cfg         Config
	session     *call.Session
	debugInfo   call.DebugInfo
	methods     list.Model
	request     viewport.Model
	response    viewport.Model
	debug       viewport.Model
	requests    map[string]string
	selected    contract.Method
	focus       paneFocus
	width       int
	height      int
	status      string
	lastError   error
	lastResult  *call.Result
	loop        *loopController
	loopStats   loopStats
	loading     bool
	reloading   bool
	wasmModTime time.Time
}

var (
	backgroundColor = lipgloss.Color("#000000")
	surfaceColor    = lipgloss.Color("#111111")
	surfaceAltColor = lipgloss.Color("#1A1A1A")
	borderColor     = lipgloss.Color("#2D2D30")
	focusColor      = lipgloss.Color("#569CD6")
	errorColor      = lipgloss.Color("#F44747")
	successColor    = lipgloss.Color("#6A9955")
	textColor       = lipgloss.Color("#D4D4D4")
	mutedColor      = lipgloss.Color("#808080")
	accentColor     = lipgloss.Color("#4EC9B0")
	warmColor       = lipgloss.Color("#DCDCAA")
	stringColor     = lipgloss.Color("#CE9178")

	appStyle = lipgloss.NewStyle().
			Padding(1).
			Background(backgroundColor).
			Foreground(textColor)
	panelStyle = lipgloss.NewStyle().
			Background(surfaceColor).
			Foreground(textColor).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)
	panelFocusStyle = panelStyle.Copy().Background(surfaceAltColor).BorderForeground(focusColor)
	panelErrorStyle = panelStyle.Copy().Background(surfaceAltColor).BorderForeground(errorColor)
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(textColor)
	subtleStyle     = lipgloss.NewStyle().Foreground(mutedColor)
	statusStyle     = lipgloss.NewStyle().Foreground(textColor)
	errorStyle      = lipgloss.NewStyle().Foreground(errorColor)
	successStyle    = lipgloss.NewStyle().Foreground(successColor)
	keyStyle        = lipgloss.NewStyle().
			Foreground(backgroundColor).
			Background(focusColor).
			Bold(true).
			Padding(0, 1)
	shortcutTextStyle = lipgloss.NewStyle().Foreground(textColor)
	topBarStyle       = lipgloss.NewStyle().
				Background(surfaceAltColor).
				Foreground(textColor).
				Padding(0, 1).
				MarginBottom(1)
	topChipStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(surfaceColor).
			Padding(0, 1).
			MarginRight(1)
	topChipAccent = topChipStyle.Copy().
			Foreground(backgroundColor).
			Background(focusColor).
			Bold(true)
	topChipSuccess = topChipStyle.Copy().
			Foreground(backgroundColor).
			Background(successColor).
			Bold(true)
	topChipMuted   = topChipStyle.Copy().Foreground(textColor).Background(borderColor)
	statusBarStyle = lipgloss.NewStyle().
			Background(surfaceColor).
			Foreground(textColor).
			Padding(0, 1).
			MarginBottom(1)
	footerBarStyle = lipgloss.NewStyle().
			Background(surfaceAltColor).
			Foreground(textColor).
			Padding(0, 1).
			MarginTop(1)
	statusChipStyle = lipgloss.NewStyle().
			Foreground(backgroundColor).
			Background(accentColor).
			Bold(true).
			Padding(0, 1).
			MarginRight(1)
	statusErrorStyle = statusChipStyle.Copy().Background(errorColor)
	statusBusyStyle  = statusChipStyle.Copy().Background(stringColor)
	infoChipStyle    = lipgloss.NewStyle().
				Foreground(textColor).
				Background(borderColor).
				Padding(0, 1).
				MarginRight(1)
)

func Run(cfg Config) error {
	if cfg.SchemaPath == "" {
		return errors.New("schema path is required")
	}
	if cfg.PluginPath == "" {
		return errors.New("plugin path is required")
	}
	session, err := newSession(cfg)
	if err != nil {
		return err
	}
	defer func() {
		_ = session.Close()
	}()

	in := cfg.Stdin
	if in == nil {
		in = os.Stdin
	}
	out := cfg.Stdout
	if out == nil {
		out = os.Stdout
	}

	p := tea.NewProgram(newModel(cfg, session), tea.WithInput(in), tea.WithOutput(out))
	_, err = p.Run()
	return err
}

func newModel(cfg Config, session *call.Session) model {
	contractModel := session.Contract()
	items := make([]list.Item, 0, len(contractModel.PluginService.Methods))
	requests := make(map[string]string, len(contractModel.PluginService.Methods))
	defaultIndex := -1
	for idx, method := range contractModel.PluginService.Methods {
		items = append(items, methodItem{method: method})
		prefill, err := session.DefaultRequestJSON(method.Name)
		if err != nil {
			prefill = "{}"
		}
		requests[method.Name] = strings.TrimRight(prefill, "\n")
		if defaultIndex == -1 && !strings.EqualFold(method.RequestType, "Empty") {
			defaultIndex = idx
		}
	}
	if defaultIndex == -1 {
		defaultIndex = 0
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(textColor)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(mutedColor)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(warmColor).
		BorderForeground(focusColor)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(textColor).
		BorderForeground(focusColor)
	delegate.Styles.DimmedTitle = delegate.Styles.DimmedTitle.Foreground(mutedColor)
	delegate.Styles.DimmedDesc = delegate.Styles.DimmedDesc.Foreground(mutedColor)

	methodList := list.New(items, delegate, 30, 12)
	methodList.Title = contractModel.Name + " Methods"
	methodList.SetShowStatusBar(false)
	methodList.SetFilteringEnabled(false)
	methodList.SetShowHelp(false)
	methodList.Select(defaultIndex)

	requestView := viewport.New(60, 12)
	responseView := viewport.New(60, 10)
	responseView.SetContent(
		"Press c to call once, l to start or stop a tight loop, or e to edit the request in your editor.",
	)
	debugView := viewport.New(60, 10)

	m := model{
		cfg:         cfg,
		session:     session,
		debugInfo:   session.DebugInfo(),
		methods:     methodList,
		request:     requestView,
		response:    responseView,
		debug:       debugView,
		requests:    requests,
		focus:       focusMethods,
		status:      "Ready",
		wasmModTime: readFileModTime(cfg.PluginPath),
	}
	m.syncSelectedMethod()
	m.renderRequest()
	m.refreshDebug()
	return m
}

func (m model) Init() tea.Cmd {
	return watchWasmCmd(m.cfg.PluginPath)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case fileCheckMsg:
		if msg.err != nil {
			m.lastError = msg.err
			m.status = "Wasm watch failed"
			m.refreshDebug()
			return m, watchWasmCmd(m.cfg.PluginPath)
		}
		if !msg.modTime.IsZero() && msg.modTime.After(m.wasmModTime) && !m.reloading && !m.loading {
			if m.loop != nil {
				stopLoop(m.loop)
				m.loop = nil
			}
			m.reloading = true
			m.status = "Detected Wasm change, reloading..."
			return m, tea.Batch(reloadSessionCmd(m.cfg), watchWasmCmd(m.cfg.PluginPath))
		}
		return m, watchWasmCmd(m.cfg.PluginPath)
	case reloadResultMsg:
		m.reloading = false
		if msg.err != nil {
			m.lastError = msg.err
			m.status = "Hot reload failed"
			m.refreshDebug()
			return m, nil
		}
		if m.session != nil {
			_ = m.session.Close()
		}
		previousMethod := m.selected.Name
		m.session = msg.session
		m.debugInfo = msg.session.DebugInfo()
		m.wasmModTime = msg.modTime
		m.lastError = nil
		m.lastResult = nil
		m.loopStats = loopStats{}
		m.loadSessionTemplates()
		m.reselectMethod(previousMethod)
		m.renderRequest()
		m.refreshDebug()
		m.status = "Hot reload succeeded"
		return m, nil
	case callResultMsg:
		m.loading = false
		m.lastError = msg.err
		if msg.err != nil {
			m.status = "Call failed for " + m.selected.Name
			m.refreshDebug()
			return m, nil
		}
		m.lastResult = &msg.result
		m.response.SetContent(strings.TrimSpace(string(msg.result.ResponseJSON)))
		m.status = fmt.Sprintf("Call succeeded for %s in %s", msg.result.Method.Name, msg.result.Duration.Truncate(time.Microsecond))
		m.refreshDebug()
		return m, nil
	case editorResultMsg:
		if msg.err != nil {
			m.lastError = msg.err
			m.status = "Editor failed"
			m.refreshDebug()
			return m, nil
		}
		m.requests[m.selected.Name] = strings.TrimRight(msg.content, "\n")
		m.renderRequest()
		m.lastError = nil
		m.status = "Loaded request from editor for " + m.selected.Name
		return m, nil
	case loopUpdateMsg:
		m.loopStats = msg.stats
		if msg.err != nil {
			m.lastError = msg.err
			m.status = fmt.Sprintf("Loop failed after %d iterations", msg.stats.Iterations)
		} else if msg.done {
			m.status = fmt.Sprintf("Loop stopped after %d iterations", msg.stats.Iterations)
		} else {
			m.status = fmt.Sprintf(
				"Loop running: %d calls @ %.0f/s avg %s",
				msg.stats.Iterations,
				callsPerSecond(msg.stats),
				averageDuration(msg.stats).Truncate(time.Microsecond),
			)
		}
		if msg.done {
			m.loop = nil
		}
		m.refreshDebug()
		if m.loop != nil {
			return m, waitLoopMsg(m.loop)
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.loop != nil {
				stopLoop(m.loop)
			}
			return m, tea.Quit
		case "tab":
			m.focus = m.focus.next()
			m.status = "Focused " + m.focus.label()
			return m, nil
		case "shift+tab":
			m.focus = m.focus.prev()
			m.status = "Focused " + m.focus.label()
			return m, nil
		case "e", "o":
			m.status = fmt.Sprintf("Opening editor for %s...", m.selected.Name)
			return m, openEditorCmd(m.requestValue())
		case "c":
			if m.loop != nil {
				m.status = "Stop the loop before running a single call"
				return m, nil
			}
			m.loading = true
			m.lastError = nil
			m.status = fmt.Sprintf("Invoking %s...", m.selected.Name)
			m.refreshDebug()
			return m, m.invokeSelectedOnce()
		case "l":
			if m.loop != nil {
				stopLoop(m.loop)
				m.status = "Stopping loop..."
				return m, nil
			}
			prepared, err := m.session.PrepareJSON(m.selected.Name, []byte(m.requestValue()))
			if err != nil {
				m.lastError = err
				m.status = fmt.Sprintf("Failed to prepare %s loop payload", m.selected.Name)
				m.refreshDebug()
				return m, nil
			}
			m.lastError = nil
			m.loopStats = loopStats{}
			m.loop = startLoop(m.session, prepared)
			m.status = "Raw loop started for " + m.selected.Name
			m.refreshDebug()
			return m, waitLoopMsg(m.loop)
		case "r":
			prefill, err := m.session.DefaultRequestJSON(m.selected.Name)
			if err != nil {
				m.lastError = err
				m.status = "Failed to build schema-derived request template"
				m.refreshDebug()
				return m, nil
			}
			m.requests[m.selected.Name] = strings.TrimRight(prefill, "\n")
			m.renderRequest()
			m.lastError = nil
			m.status = fmt.Sprintf("Reset request for %s from schema", m.selected.Name)
			return m, nil
		case "p":
			m.requests[m.selected.Name] = prettyInput(m.requestValue())
			m.renderRequest()
			m.status = "Prettified request JSON"
			return m, nil
		}
		return m.handleFocusedKey(msg)
	}
	return m, nil
}

func (m model) View() string {
	methodTitle := m.renderPanelTitle("Methods", focusMethods)
	requestTitle := m.renderPanelTitle("Request", focusRequest)
	responseTitle := m.renderPanelTitle("Response", focusResponse)
	debugTitle := m.renderPanelTitle("Debug", focusDebug)

	leftWidth := max(28, m.width/4)
	rightWidth := max(50, m.width-leftWidth-6)

	leftPanel := m.panelFor(focusMethods).
		Width(leftWidth).
		Render(methodTitle + "\n" + m.methods.View())
	reqPanel := m.panelFor(focusRequest).
		Width(rightWidth).
		Render(requestTitle + "\n" + m.request.View())

	responseBody := m.response.View()
	responsePanelStyle := panelStyle
	if m.lastError != nil {
		responseTitle = errorStyle.Render("Error")
		responseBody = errorStyle.Render(m.lastError.Error())
		responsePanelStyle = panelErrorStyle
	} else if m.loading {
		responseBody = subtleStyle.Render("Calling plugin...")
	} else if m.loop != nil {
		responseBody = successStyle.Render("Loop running. Press l to stop.")
	} else if m.reloading {
		responseBody = subtleStyle.Render("Reloading plugin after file change...")
	}
	if m.lastError == nil {
		responsePanelStyle = m.panelFor(focusResponse)
	}
	respPanel := responsePanelStyle.Width(rightWidth).Render(responseTitle + "\n" + responseBody)
	debugPanel := m.panelFor(focusDebug).
		Width(rightWidth).
		Render(debugTitle + "\n" + m.debug.View())

	return appStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderTopBar(),
		m.renderStatusBar(),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftPanel,
			lipgloss.JoinVertical(lipgloss.Left, reqPanel, respPanel, debugPanel),
		),
		m.renderFooter(),
	))
}

func (m *model) renderTopBar() string {
	parts := []string{
		topChipAccent.Render("Hookr TUI"),
		topChipStyle.Render("schema: " + filepath.Base(m.debugInfo.SchemaPath)),
		topChipStyle.Render("plugin: " + filepath.Base(m.debugInfo.PluginPath)),
		topChipStyle.Render("method: " + zeroOrValue(m.selected.Name)),
	}
	if m.lastResult != nil {
		parts = append(
			parts,
			topChipSuccess.Render(
				"last: "+m.lastResult.Duration.Truncate(time.Microsecond).String(),
			),
		)
	}
	if m.loop != nil || m.loopStats.Iterations > 0 {
		parts = append(
			parts,
			topChipSuccess.Render(
				fmt.Sprintf(
					"loop: %d @ %s",
					m.loopStats.Iterations,
					averageDuration(m.loopStats).Truncate(time.Microsecond),
				),
			),
			topChipSuccess.Render(fmt.Sprintf("cps: %.0f", callsPerSecond(m.loopStats))),
		)
	}
	if m.reloading {
		parts = append(parts, topChipMuted.Render("reloading"))
	}
	return topBarStyle.Width(max(0, m.width-2)).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, parts...))
}

func (m *model) renderPanelTitle(title string, pane paneFocus) string {
	if m.focus == pane {
		return titleStyle.Copy().Foreground(focusColor).Render(title + " *")
	}
	return titleStyle.Render(title)
}

func (m *model) renderStatusBar() string {
	parts := []string{
		m.renderStatusChip(),
		infoChipStyle.Render("focus: " + m.focus.label()),
	}
	if m.lastResult != nil {
		parts = append(parts,
			infoChipStyle.Render("last req: "+fmt.Sprintf("%dB", m.lastResult.RequestBytes)),
			infoChipStyle.Render("last resp: "+fmt.Sprintf("%dB", m.lastResult.ResponseBytes)),
		)
	}
	if m.loop != nil || m.loopStats.Iterations > 0 {
		parts = append(
			parts,
			infoChipStyle.Render(fmt.Sprintf("calls/sec: %.0f", callsPerSecond(m.loopStats))),
			infoChipStyle.Render(
				"avg: "+averageDuration(m.loopStats).Truncate(time.Microsecond).String(),
			),
			infoChipStyle.Render("min: "+displayDuration(m.loopStats.Min).String()),
			infoChipStyle.Render("max: "+displayDuration(m.loopStats.Max).String()),
		)
	}
	return statusBarStyle.Width(max(0, m.width-2)).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, parts...))
}

func (m *model) renderStatusChip() string {
	switch {
	case m.lastError != nil:
		return statusErrorStyle.Render(m.status)
	case m.loading || m.reloading || m.loop != nil:
		return statusBusyStyle.Render(m.status)
	default:
		return statusChipStyle.Render(m.status)
	}
}

func (m *model) panelFor(pane paneFocus) lipgloss.Style {
	if m.focus == pane {
		return panelFocusStyle
	}
	return panelStyle
}

func (m *model) renderFooter() string {
	shortcuts := strings.Join([]string{
		renderShortcut("up/down", "move"),
		renderShortcut("tab", "panes"),
		renderShortcut("e", "editor"),
		renderShortcut("c", "call"),
		renderShortcut("l", "loop"),
		renderShortcut("r", "reset"),
		renderShortcut("p", "pretty"),
		renderShortcut("q", "quit"),
	}, "  ")
	return footerBarStyle.Width(max(0, m.width-2)).Render(
		statusStyle.Render(shortcuts),
	)
}

func renderShortcut(key string, label string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		keyStyle.Render(key),
		shortcutTextStyle.Render(" "+label),
	)
}

func (m *model) syncSelectionChange() {
	before := m.selected.Name
	m.syncSelectedMethod()
	if m.selected.Name != before {
		m.renderRequest()
		m.status = "Selected " + m.selected.Name
		m.refreshDebug()
	}
}

func (m *model) syncSelectedMethod() {
	item, ok := m.methods.SelectedItem().(methodItem)
	if !ok {
		m.selected = contract.Method{}
		return
	}
	m.selected = item.method
}

func (m model) handleFocusedKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusMethods:
		var cmd tea.Cmd
		m.methods, cmd = m.methods.Update(msg)
		m.syncSelectionChange()
		return m, cmd
	case focusRequest:
		var cmd tea.Cmd
		m.request, cmd = m.request.Update(msg)
		return m, cmd
	case focusResponse:
		var cmd tea.Cmd
		m.response, cmd = m.response.Update(msg)
		return m, cmd
	case focusDebug:
		var cmd tea.Cmd
		m.debug, cmd = m.debug.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m *model) requestValue() string {
	value := m.requests[m.selected.Name]
	if value == "" {
		return "{}"
	}
	return value
}

func (m *model) renderRequest() {
	m.request.SetContent(m.requestValue())
}

func (m *model) invokeSelectedOnce() tea.Cmd {
	methodName := m.selected.Name
	requestJSON := []byte(m.requestValue())
	return func() tea.Msg {
		result, err := m.session.InvokeJSON(context.Background(), methodName, requestJSON)
		return callResultMsg{result: result, err: err}
	}
}

func (m *model) refreshDebug() {
	lines := []string{
		subtleStyle.Render("OVERVIEW"),
		debugRow("method", fmt.Sprintf("%s (id=%d)", m.selected.Name, m.selected.ID)),
		debugRow("types", fmt.Sprintf("%s -> %s", m.selected.RequestType, m.selected.ResponseType)),
		debugRow("focus", m.focus.label()),
		"",
		subtleStyle.Render("RUNTIME"),
		debugRow("schema", m.debugInfo.SchemaPath),
		debugRow("plugin", m.debugInfo.PluginPath),
		debugRow("abi", zeroOrValue(m.debugInfo.ABIVersion)),
		debugRow("caps", fmt.Sprintf("0x%x", m.debugInfo.Capabilities)),
		debugRow("expected hash", m.debugInfo.SchemaHash),
		debugRow("plugin hash", zeroOrValue(m.debugInfo.PluginSchemaHash)),
		debugRow("hash match", strconv.FormatBool(m.debugInfo.SchemaHashMatch)),
		debugRow(
			"methods",
			fmt.Sprintf(
				"%d / %d",
				len(m.debugInfo.PluginMethodIDs),
				m.debugInfo.ContractMethodCount,
			),
		),
	}
	if m.cfg.HostFixturePath != "" {
		lines = append(lines, debugRow("host fixture", m.cfg.HostFixturePath))
	}
	if m.lastResult != nil {
		lines = append(lines,
			"",
			subtleStyle.Render("LAST CALL"),
			debugRow("duration", m.lastResult.Duration.Truncate(time.Microsecond).String()),
			debugRow("request", fmt.Sprintf("%dB", m.lastResult.RequestBytes)),
			debugRow("response", fmt.Sprintf("%dB", m.lastResult.ResponseBytes)),
		)
	}
	if m.loop != nil || m.loopStats.Iterations > 0 {
		lines = append(
			lines,
			"",
			subtleStyle.Render("LOOP"),
			debugRow("iterations", strconv.Itoa(m.loopStats.Iterations)),
			debugRow("errors", strconv.Itoa(m.loopStats.Errors)),
			debugRow("calls/sec", fmt.Sprintf("%.2f", callsPerSecond(m.loopStats))),
			debugRow("avg", averageDuration(m.loopStats).Truncate(time.Microsecond).String()),
			debugRow("min", displayDuration(m.loopStats.Min).String()),
			debugRow("max", displayDuration(m.loopStats.Max).String()),
			debugRow("last", displayDuration(m.loopStats.Last).String()),
			debugRow("elapsed", m.loopStats.Elapsed.Truncate(time.Millisecond).String()),
			debugRow(
				"req/resp",
				fmt.Sprintf("%dB / %dB", m.loopStats.RequestBytes, m.loopStats.ResponseBytes),
			),
			debugRow("req total", formatBytes(m.loopStats.TotalRequestBytes)),
			debugRow("resp total", formatBytes(m.loopStats.TotalResponseBytes)),
			debugRow(
				"req/sec",
				formatBytesPerSecond(
					bytesPerSecond(m.loopStats.TotalRequestBytes, m.loopStats.Elapsed),
				),
			),
			debugRow(
				"resp/sec",
				formatBytesPerSecond(
					bytesPerSecond(m.loopStats.TotalResponseBytes, m.loopStats.Elapsed),
				),
			),
		)
	}
	m.debug.SetContent(strings.Join(lines, "\n"))
}

func debugRow(label string, value string) string {
	return fmt.Sprintf("%-12s %s", label, value)
}

func (m *model) resize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	leftWidth := max(28, m.width/4)
	rightWidth := max(50, m.width-leftWidth-8)
	bodyHeight := max(14, m.height-8)
	requestHeight := max(8, bodyHeight/3)
	responseHeight := max(6, bodyHeight/3)
	debugHeight := max(6, bodyHeight-requestHeight-responseHeight-4)

	m.methods.SetSize(leftWidth-4, bodyHeight)
	m.request.Width = rightWidth - 4
	m.request.Height = requestHeight
	m.response.Width = rightWidth - 4
	m.response.Height = responseHeight
	m.debug.Width = rightWidth - 4
	m.debug.Height = debugHeight
}

func startLoop(session *call.Session, prepared call.PreparedCall) *loopController {
	controller := &loopController{
		stop:    make(chan struct{}),
		updates: make(chan loopUpdateMsg, 8),
	}
	go func() {
		defer close(controller.updates)
		stats := loopStats{Started: time.Now()}
		lastEmit := time.Now()
		for {
			select {
			case <-controller.stop:
				stats.Elapsed = time.Since(stats.Started)
				controller.updates <- loopUpdateMsg{stats: stats, done: true}
				return
			default:
			}

			result, err := session.InvokePrepared(context.Background(), prepared)
			if err != nil {
				stats.Errors++
				stats.Elapsed = time.Since(stats.Started)
				controller.updates <- loopUpdateMsg{stats: stats, err: err, done: true}
				return
			}
			stats.Iterations++
			stats.Last = result.Duration
			stats.Total += result.Duration
			stats.Elapsed = time.Since(stats.Started)
			stats.RequestBytes = result.RequestBytes
			stats.ResponseBytes = result.ResponseBytes
			stats.TotalRequestBytes += int64(result.RequestBytes)
			stats.TotalResponseBytes += int64(result.ResponseBytes)
			if stats.Min == 0 || result.Duration < stats.Min {
				stats.Min = result.Duration
			}
			if result.Duration > stats.Max {
				stats.Max = result.Duration
			}
			if stats.Iterations == 1 || time.Since(lastEmit) >= 100*time.Millisecond {
				lastEmit = time.Now()
				controller.updates <- loopUpdateMsg{stats: stats}
			}
		}
	}()
	return controller
}

func waitLoopMsg(controller *loopController) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-controller.updates
		if !ok {
			return loopUpdateMsg{done: true}
		}
		return msg
	}
}

func stopLoop(controller *loopController) {
	if controller == nil {
		return
	}
	select {
	case <-controller.stop:
	default:
		close(controller.stop)
	}
}

func openEditorCmd(content string) tea.Cmd {
	tmp, err := os.CreateTemp("", "hookr-request-*.json")
	if err != nil {
		return func() tea.Msg { return editorResultMsg{err: err} }
	}
	path := tmp.Name()
	_ = tmp.Close()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		_ = os.Remove(path)
		return func() tea.Msg { return editorResultMsg{err: err} }
	}
	cmd, err := editorCommand(path)
	if err != nil {
		_ = os.Remove(path)
		return func() tea.Msg { return editorResultMsg{err: err} }
	}
	return tea.ExecProcess(cmd, func(execErr error) tea.Msg {
		defer func() { _ = os.Remove(path) }()
		updated, readErr := os.ReadFile(path)
		if readErr != nil {
			return editorResultMsg{err: readErr}
		}
		if execErr != nil {
			return editorResultMsg{content: string(updated), err: execErr}
		}
		return editorResultMsg{content: string(updated)}
	})
}

func editorCommand(path string) (*exec.Cmd, error) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vi"
		}
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return nil, errors.New("editor command is empty")
	}
	parts = append(parts, path)
	return exec.Command(parts[0], parts[1:]...), nil
}

func prettyInput(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "{}"
	}
	var value any
	if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
		return raw
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return raw
	}
	return string(encoded)
}

func averageDuration(stats loopStats) time.Duration {
	if stats.Iterations == 0 {
		return 0
	}
	return time.Duration(int64(stats.Total) / int64(stats.Iterations))
}

func callsPerSecond(stats loopStats) float64 {
	if stats.Iterations == 0 || stats.Elapsed <= 0 {
		return 0
	}
	return float64(stats.Iterations) / stats.Elapsed.Seconds()
}

func bytesPerSecond(total int64, elapsed time.Duration) float64 {
	if total <= 0 || elapsed <= 0 {
		return 0
	}
	return float64(total) / elapsed.Seconds()
}

func formatBytes(total int64) string {
	return formatRate(float64(total))
}

func formatBytesPerSecond(rate float64) string {
	return formatRate(rate)
}

func formatRate(value float64) string {
	units := []string{"B", "KiB", "MiB", "GiB"}
	unit := units[0]
	for i := 1; i < len(units) && value >= 1024; i++ {
		value /= 1024
		unit = units[i]
	}
	if unit == "B" {
		return fmt.Sprintf("%.0f %s", value, unit)
	}
	return fmt.Sprintf("%.2f %s", value, unit)
}

func displayDuration(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return d.Truncate(time.Microsecond)
}

func zeroOrValue(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func watchWasmCmd(path string) tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		info, err := os.Stat(path)
		if err != nil {
			return fileCheckMsg{err: err}
		}
		return fileCheckMsg{modTime: info.ModTime()}
	})
}

func reloadSessionCmd(cfg Config) tea.Cmd {
	return func() tea.Msg {
		session, err := newSession(cfg)
		if err != nil {
			return reloadResultMsg{err: err}
		}
		return reloadResultMsg{session: session, modTime: readFileModTime(cfg.PluginPath)}
	}
}

func readFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func (m *model) loadSessionTemplates() {
	for _, method := range m.session.Contract().PluginService.Methods {
		if _, ok := m.requests[method.Name]; ok {
			continue
		}
		prefill, err := m.session.DefaultRequestJSON(method.Name)
		if err != nil {
			prefill = "{}"
		}
		m.requests[method.Name] = strings.TrimRight(prefill, "\n")
	}
}

func (m *model) reselectMethod(name string) {
	for idx, item := range m.methods.Items() {
		mi, ok := item.(methodItem)
		if !ok {
			continue
		}
		if mi.method.Name == name {
			m.methods.Select(idx)
			m.selected = mi.method
			return
		}
	}
	m.syncSelectedMethod()
}

func (p paneFocus) next() paneFocus {
	switch p {
	case focusMethods:
		return focusRequest
	case focusRequest:
		return focusResponse
	case focusResponse:
		return focusDebug
	default:
		return focusMethods
	}
}

func (p paneFocus) prev() paneFocus {
	switch p {
	case focusMethods:
		return focusDebug
	case focusRequest:
		return focusMethods
	case focusResponse:
		return focusRequest
	default:
		return focusResponse
	}
}

func (p paneFocus) label() string {
	switch p {
	case focusMethods:
		return "methods"
	case focusRequest:
		return "request"
	case focusResponse:
		return "response"
	case focusDebug:
		return "debug"
	default:
		return "pane"
	}
}

func newSession(cfg Config) (*call.Session, error) {
	return call.NewSession(call.Config{
		SchemaPath:        cfg.SchemaPath,
		PluginPath:        cfg.PluginPath,
		HostFixturePath:   cfg.HostFixturePath,
		Hash:              cfg.Hash,
		AllowUnsigned:     cfg.AllowUnsigned,
		FlatcPath:         cfg.FlatcPath,
		IncludePaths:      cfg.IncludePaths,
		Package:           cfg.Package,
		PluginService:     cfg.PluginService,
		OptionalAttribute: cfg.OptionalAttribute,
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
