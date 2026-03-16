package runtime

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
)

const defaultReloadDebounce = 250 * time.Millisecond

// Invoker is the minimal runtime surface used by generated host SDKs and
// reload hooks.
type Invoker interface {
	InvokeMethod(context.Context, uint32, []byte) ([]byte, error)
	InvokeMethodWithResponse(context.Context, uint32, []byte, func([]byte) error) error
	HasPluginMethodID(uint32) bool
	PluginHandshake() (runtimecontract.Handshake, bool)
	Close(context.Context) error
}

// ReloadConfig configures automatic live reload for a plugin runtime.
type ReloadConfig struct {
	Debounce      time.Duration
	OnReload      func(context.Context, Invoker, ReloadEvent) error
	OnReloadError func(context.Context, error)
}

// ReloadEvent describes a successful live reload attempt.
type ReloadEvent struct {
	PluginPath string
	Time       time.Time
	Previous   RuntimeInfo
	Current    RuntimeInfo
}

// RuntimeInfo captures the relevant runtime metadata for reload hooks.
type RuntimeInfo struct {
	SchemaHash   [runtimecontract.SchemaHashLen]byte
	ABIMajor     uint16
	ABIMinor     uint16
	Capabilities uint64
	MethodIDs    []uint32
}

// LiveRuntime wraps Runtime with file watching and atomic reload behavior.
type LiveRuntime struct {
	ctx        context.Context
	opts       []Option
	pluginPath string
	reload     ReloadConfig

	mu      sync.RWMutex
	current *Runtime
	closed  bool

	watcher *fsnotify.Watcher
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// NewLive creates a plugin runtime that watches the plugin file and reloads it
// when the artifact changes on disk.
func NewLive(ctx context.Context, cfg ReloadConfig, opts ...Option) (*LiveRuntime, error) {
	current, err := New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	if current.file == nil || current.file.Path == "" {
		_ = current.Close(ctx)
		return nil, errors.New("plugin file not configured")
	}

	pluginPath, err := filepath.Abs(current.file.Path)
	if err != nil {
		_ = current.Close(ctx)
		return nil, fmt.Errorf("resolve plugin path: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		_ = current.Close(ctx)
		return nil, fmt.Errorf("create file watcher: %w", err)
	}
	if err := watcher.Add(filepath.Dir(pluginPath)); err != nil {
		_ = watcher.Close()
		_ = current.Close(ctx)
		return nil, fmt.Errorf("watch plugin directory: %w", err)
	}

	live := &LiveRuntime{
		ctx:        ctx,
		opts:       slices.Clone(opts),
		pluginPath: pluginPath,
		reload:     cfg,
		current:    current,
		watcher:    watcher,
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
	}
	go live.watch()
	return live, nil
}

func (l *LiveRuntime) InvokeMethod(
	ctx context.Context,
	methodID uint32,
	payload []byte,
) ([]byte, error) {
	l.mu.RLock()
	rt, err := l.currentLocked()
	if err != nil {
		l.mu.RUnlock()
		return nil, err
	}
	defer l.mu.RUnlock()
	return rt.InvokeMethod(ctx, methodID, payload)
}

func (l *LiveRuntime) InvokeMethodWithResponse(
	ctx context.Context,
	methodID uint32,
	payload []byte,
	fn func([]byte) error,
) error {
	l.mu.RLock()
	rt, err := l.currentLocked()
	if err != nil {
		l.mu.RUnlock()
		return err
	}
	defer l.mu.RUnlock()
	return rt.InvokeMethodWithResponse(ctx, methodID, payload, fn)
}

func (l *LiveRuntime) HasPluginMethodID(methodID uint32) bool {
	l.mu.RLock()
	rt, err := l.currentLocked()
	if err != nil {
		l.mu.RUnlock()
		return false
	}
	defer l.mu.RUnlock()
	return rt.HasPluginMethodID(methodID)
}

func (l *LiveRuntime) PluginHandshake() (runtimecontract.Handshake, bool) {
	l.mu.RLock()
	rt, err := l.currentLocked()
	if err != nil {
		l.mu.RUnlock()
		return runtimecontract.Handshake{}, false
	}
	defer l.mu.RUnlock()
	return rt.PluginHandshake()
}

func (l *LiveRuntime) Close(ctx context.Context) error {
	select {
	case <-l.doneCh:
	default:
		close(l.stopCh)
		<-l.doneCh
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	if l.current == nil {
		return nil
	}
	current := l.current
	l.current = nil
	return current.Close(ctx)
}

func (l *LiveRuntime) watch() {
	defer close(l.doneCh)
	defer func() {
		_ = l.watcher.Close()
	}()

	debounce := l.reload.Debounce
	if debounce <= 0 {
		debounce = defaultReloadDebounce
	}

	var (
		timer  *time.Timer
		timerC <-chan time.Time
	)

	resetTimer := func() {
		if timer == nil {
			timer = time.NewTimer(debounce)
			timerC = timer.C
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(debounce)
		timerC = timer.C
	}

	for {
		select {
		case <-l.stopCh:
			if timer != nil {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
			}
			return
		case <-l.ctx.Done():
			return
		case event, ok := <-l.watcher.Events:
			if !ok {
				return
			}
			if !l.matchesPluginEvent(event) {
				continue
			}
			resetTimer()
		case err, ok := <-l.watcher.Errors:
			if !ok {
				return
			}
			l.reportReloadError(fmt.Errorf("watch plugin: %w", err))
		case <-timerC:
			timerC = nil
			if err := l.reloadNow(); err != nil {
				l.reportReloadError(err)
			}
		}
	}
}

func (l *LiveRuntime) matchesPluginEvent(event fsnotify.Event) bool {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
		return false
	}
	name, err := filepath.Abs(event.Name)
	if err != nil {
		return false
	}
	return name == l.pluginPath
}

func (l *LiveRuntime) reloadNow() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	current, err := l.currentLocked()
	if err != nil {
		return err
	}

	next, err := New(l.ctx, l.opts...)
	if err != nil {
		return fmt.Errorf("reload plugin: %w", err)
	}

	event := ReloadEvent{
		PluginPath: l.pluginPath,
		Time:       time.Now(),
		Previous:   runtimeInfo(current),
		Current:    runtimeInfo(next),
	}
	if l.reload.OnReload != nil {
		if err := l.reload.OnReload(l.ctx, next, event); err != nil {
			_ = next.Close(l.ctx)
			return fmt.Errorf("reload hook failed: %w", err)
		}
	}

	l.current = next
	if err := current.Close(l.ctx); err != nil {
		return fmt.Errorf("close previous runtime after reload: %w", err)
	}
	return nil
}

func (l *LiveRuntime) currentLocked() (*Runtime, error) {
	if l.closed || l.current == nil {
		return nil, errors.New("plugin not initialized")
	}
	return l.current, nil
}

func (l *LiveRuntime) reportReloadError(err error) {
	if err == nil || l.reload.OnReloadError == nil {
		return
	}
	l.reload.OnReloadError(l.ctx, err)
}

func runtimeInfo(rt *Runtime) RuntimeInfo {
	info := RuntimeInfo{}
	if rt == nil {
		return info
	}
	if hs, ok := rt.PluginHandshake(); ok {
		info.SchemaHash = hs.SchemaHash
		info.ABIMajor = hs.ABIMajor
		info.ABIMinor = hs.ABIMinor
		info.Capabilities = hs.Capabilities
	}
	if len(rt.pluginMethods) == 0 {
		return info
	}
	info.MethodIDs = make([]uint32, 0, len(rt.pluginMethods))
	for methodID := range rt.pluginMethods {
		info.MethodIDs = append(info.MethodIDs, methodID)
	}
	slices.Sort(info.MethodIDs)
	return info
}
