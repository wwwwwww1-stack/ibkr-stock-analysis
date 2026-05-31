package capture

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var ErrUnsupported = errors.New("interactive chart screenshot capture is only supported on macOS")

type ChartCapture struct {
	Path       string
	MimeType   string
	Source     string
	CapturedAt time.Time
}

type ChartCapturer interface {
	CaptureChart(ctx context.Context, target WindowTarget) (ChartCapture, error)
}

type ChartWindowLister interface {
	ListWindows(ctx context.Context) ([]WindowInfo, error)
}

type WindowTarget struct {
	ID      int
	AppName string
	Title   string
}

type WindowInfo struct {
	ID      int
	AppName string
	Title   string
}

type InteractiveCapturer struct {
	Command         string
	TempDir         string
	Now             func() time.Time
	ListWindowsFunc func(ctx context.Context) ([]WindowInfo, error)
}

func NewInteractiveCapturer() *InteractiveCapturer {
	return &InteractiveCapturer{Command: "/usr/sbin/screencapture"}
}

func (c *InteractiveCapturer) ListWindows(ctx context.Context) ([]WindowInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if runtime.GOOS != "darwin" {
		return nil, ErrUnsupported
	}
	if c.ListWindowsFunc != nil {
		return sanitizeWindows(c.ListWindowsFunc(ctx))
	}
	return platformListWindows(ctx)
}

func (c *InteractiveCapturer) CaptureChart(ctx context.Context, target WindowTarget) (ChartCapture, error) {
	if err := ctx.Err(); err != nil {
		return ChartCapture{}, err
	}
	if runtime.GOOS != "darwin" {
		return ChartCapture{}, ErrUnsupported
	}
	command := strings.TrimSpace(c.Command)
	if command == "" {
		command = "/usr/sbin/screencapture"
	}
	file, err := os.CreateTemp(c.TempDir, "ibkr-chart-*.png")
	if err != nil {
		return ChartCapture{}, fmt.Errorf("create screenshot temp file: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return ChartCapture{}, fmt.Errorf("close screenshot temp file: %w", err)
	}

	args := []string{"-i", "-x", "-t", "png", path}
	source := "desktop_region"
	if hasWindowTarget(target) {
		window, err := c.resolveWindow(ctx, target)
		if err != nil {
			_ = os.Remove(path)
			return ChartCapture{}, err
		}
		args = []string{"-l", strconv.Itoa(window.ID), "-x", "-t", "png", path}
		source = "window:" + window.DisplayName()
	}

	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(path)
		details := strings.TrimSpace(string(output))
		if details == "" {
			details = err.Error()
		}
		return ChartCapture{}, fmt.Errorf("screenshot capture cancelled or failed: %s", details)
	}
	now := time.Now().UTC()
	if c.Now != nil {
		now = c.Now().UTC()
	}
	return ChartCapture{
		Path:       path,
		MimeType:   "image/png",
		Source:     source,
		CapturedAt: now,
	}, nil
}

func (c *InteractiveCapturer) resolveWindow(ctx context.Context, target WindowTarget) (WindowInfo, error) {
	windows, err := c.ListWindows(ctx)
	if err != nil {
		return WindowInfo{}, err
	}
	target.AppName = strings.TrimSpace(target.AppName)
	target.Title = strings.TrimSpace(target.Title)
	for _, window := range windows {
		if target.ID > 0 && window.ID == target.ID && matchesWindowTarget(window, target) {
			return window, nil
		}
	}
	var appMatch *WindowInfo
	for i := range windows {
		window := windows[i]
		if target.AppName != "" && !strings.EqualFold(window.AppName, target.AppName) {
			continue
		}
		if target.Title != "" && strings.EqualFold(window.Title, target.Title) {
			return window, nil
		}
		if appMatch == nil {
			appMatch = &window
		}
	}
	if appMatch != nil {
		return *appMatch, nil
	}
	return WindowInfo{}, fmt.Errorf("selected chart window not found: %s", target.DisplayName())
}

func matchesWindowTarget(window WindowInfo, target WindowTarget) bool {
	if target.AppName != "" && !strings.EqualFold(window.AppName, target.AppName) {
		return false
	}
	if target.Title != "" && !strings.EqualFold(window.Title, target.Title) {
		return false
	}
	return true
}

func hasWindowTarget(target WindowTarget) bool {
	return target.ID > 0 || strings.TrimSpace(target.AppName) != "" || strings.TrimSpace(target.Title) != ""
}

func sanitizeWindows(windows []WindowInfo, err error) ([]WindowInfo, error) {
	if err != nil {
		return nil, err
	}
	out := make([]WindowInfo, 0, len(windows))
	seen := make(map[int]struct{}, len(windows))
	for _, window := range windows {
		window.AppName = strings.TrimSpace(window.AppName)
		window.Title = strings.TrimSpace(window.Title)
		if window.ID <= 0 || window.AppName == "" {
			continue
		}
		if _, ok := seen[window.ID]; ok {
			continue
		}
		seen[window.ID] = struct{}{}
		out = append(out, window)
	}
	return out, nil
}

func (w WindowInfo) DisplayName() string {
	if strings.TrimSpace(w.Title) == "" {
		return strings.TrimSpace(w.AppName)
	}
	return strings.TrimSpace(w.AppName) + " - " + strings.TrimSpace(w.Title)
}

func (t WindowTarget) DisplayName() string {
	app := strings.TrimSpace(t.AppName)
	title := strings.TrimSpace(t.Title)
	if app == "" && title == "" && t.ID > 0 {
		return "window " + strconv.Itoa(t.ID)
	}
	if title == "" {
		return app
	}
	if app == "" {
		return title
	}
	return app + " - " + title
}
