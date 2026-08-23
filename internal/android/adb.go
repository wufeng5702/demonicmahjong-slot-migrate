package android

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed embedded/adb.exe embedded/AdbWinApi.dll embedded/AdbWinUsbApi.dll
var adbFS embed.FS

// ADBVersion is the platform-tools version baked into the embedded binaries.
// It is used as a cache key: bumping it forces a re-extraction on existing installs.
const ADBVersion = "34.0.5"

// GameDBPaths lists the on-device candidate paths of the save database, ordered
// by priority. The standard install path is tried first; TapTap's sandboxed
// layout is tried as a fallback (TapTap wraps the game data dir under its own
// package directory).
var GameDBPaths = []string{
	"/sdcard/Android/data/com.itaotuo.wodima/files/game.db",
	"/sdcard/Android/data/com.taptap/files/tap_sandbox_sd/0/Android/data/com.itaotuo.wodima/files/game.db",
}

// Sentinel errors for structured UI handling.
var (
	ErrNoDevice        = errors.New("no adb device connected")
	ErrUnauthorized    = errors.New("device is not authorized; allow USB debugging on the phone")
	ErrPathNotFound    = errors.New("save file not found on device; is the game installed and has it created a save?")
	ErrUsbDebuggingOff = errors.New("cannot reach adb daemon; verify USB debugging is enabled and reconnect the device")
)

// Device is one entry reported by `adb devices`.
type Device struct {
	Serial string `json:"serial"`
	State  string `json:"state"`
}

// EnsureADB extracts the embedded adb.exe and its helper DLLs to a per-user
// cache directory. The directory is keyed by ADBVersion so an upgrade replaces
// the files deterministically. It returns the absolute path of adb.exe.
func EnsureADB() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	if base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "wodima-migrate", "adb-"+ADBVersion)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	adbExe := filepath.Join(dir, "adb.exe")
	for _, name := range []string{"adb.exe", "AdbWinApi.dll", "AdbWinUsbApi.dll"} {
		dst := filepath.Join(dir, name)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		data, err := adbFS.ReadFile("embedded/" + name)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return "", err
		}
	}
	return adbExe, nil
}

var adbDeviceRe = regexp.MustCompile(`^(\S+)\s+(\S+)`)

// ListDevices runs `adb devices` and returns the parsed device list.
// It starts the daemon if necessary; a failure to read any device typically
// means USB debugging is off or no device is connected.
func ListDevices(adbPath string) ([]Device, error) {
	out, err := exec.Command(adbPath, "start-server").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUsbDebuggingOff, strings.TrimSpace(string(out)))
	}
	out, err = exec.Command(adbPath, "devices").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUsbDebuggingOff, strings.TrimSpace(string(out)))
	}
	var devices []Device
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices") || strings.HasPrefix(line, "*") {
			continue
		}
		m := adbDeviceRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		devices = append(devices, Device{Serial: m[1], State: m[2]})
	}
	return devices, nil
}

// PulledDB is one successfully pulled game.db file.
type PulledDB struct {
	SrcPath string `json:"sourcePath"` // on-device path it was pulled from
	DstPath string `json:"path"`       // local destination path
}

// PullAllGameDBs pulls every existing game.db candidate from the device into
// dstDir. Each candidate gets its own local file (game_0.db, game_1.db, ...).
// Missing paths are skipped silently; other errors abort the operation.
// Returns one entry per successful pull; if none succeeded returns an error.
func PullAllGameDBs(adbPath, deviceSerial, dstDir string) ([]PulledDB, error) {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return nil, err
	}

	var result []PulledDB
	var lastErr error
	for i, src := range GameDBPaths {
		dst := filepath.Join(dstDir, fmt.Sprintf("game_%d.db", i))
		err := pullOnce(adbPath, deviceSerial, src, dst)
		if err == nil {
			result = append(result, PulledDB{SrcPath: src, DstPath: dst})
			continue
		}
		if errors.Is(err, ErrPathNotFound) {
			lastErr = err
			continue
		}
		// Unauthorized, no device, or real failure: abort.
		return nil, err
	}
	if len(result) == 0 {
		if lastErr == nil {
			lastErr = ErrPathNotFound
		}
		return nil, lastErr
	}
	return result, nil
}

// pullOnce runs a single `adb pull` for the given source path.
func pullOnce(adbPath, deviceSerial, srcPath, dstPath string) error {
	cmd := exec.Command(adbPath, "-s", deviceSerial, "pull", srcPath, dstPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = nil
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		switch {
		case strings.Contains(msg, "does not exist") || strings.Contains(msg, "No such file") || strings.Contains(msg, "remote object"):
			return fmt.Errorf("%w: %s", ErrPathNotFound, strings.TrimSpace(msg))
		case strings.Contains(msg, "unauthorized"):
			return fmt.Errorf("%w: %s", ErrUnauthorized, strings.TrimSpace(msg))
		case strings.Contains(msg, "device") && strings.Contains(msg, "not found"):
			return fmt.Errorf("%w: %s", ErrNoDevice, strings.TrimSpace(msg))
		default:
			return fmt.Errorf("adb pull failed: %s: %v", strings.TrimSpace(msg), err)
		}
	}
	if _, err := os.Stat(dstPath); err != nil {
		return fmt.Errorf("adb pull reported success but destination is missing: %w", err)
	}
	return nil
}

// Compile-time assertion that embed.FS is used.
var _ fs.FS = (*embed.FS)(nil)
