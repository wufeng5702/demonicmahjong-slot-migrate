package android

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Service struct {
	app *application.App
}

func NewService(app *application.App) *Service {
	return &Service{app: app}
}

// AutoFetchAndroidDBRequest carries arguments for AutoFetchAndroidDB.
type AutoFetchAndroidDBRequest struct{}

// AutoFetchAndroidDBResponse is the return envelope for AutoFetchAndroidDB.
// Files contains one entry per on-device save that was successfully pulled
// (e.g. both the standard install and the TapTap sandbox install).
type AutoFetchAndroidDBResponse struct {
	Files []PulledDB `json:"files"`
}

// AutoFetchAndroidDB extracts the bundled adb, locates a connected device and
// pulls every existing game.db candidate into a per-user cache directory.
// Returns the local path and the on-device source path for each pulled file.
func (s *Service) AutoFetchAndroidDB(req *AutoFetchAndroidDBRequest) (*AutoFetchAndroidDBResponse, error) {
	_ = req
	adbPath, err := EnsureADB()
	if err != nil {
		return nil, fmt.Errorf("prepare adb: %w", err)
	}
	devices, err := ListDevices(adbPath)
	if err != nil {
		return nil, err
	}
	var serial string
	for _, d := range devices {
		if d.State == "device" {
			serial = d.Serial
			break
		}
	}
	if serial == "" {
		if len(devices) > 0 {
			return nil, ErrUnauthorized
		}
		return nil, ErrNoDevice
	}
	dstDir, err := cachedGameDBDir()
	if err != nil {
		return nil, err
	}
	files, err := PullAllGameDBs(adbPath, serial, dstDir)
	if err != nil {
		return nil, err
	}
	return &AutoFetchAndroidDBResponse{Files: files}, nil
}

// ReadAndroidSlotsRequest carries arguments for ReadAndroidSlots.
type ReadAndroidSlotsRequest struct {
	DBPath string `json:"dbPath"`
}

// ReadAndroidSlotsResponse is the return envelope for ReadAndroidSlots.
type ReadAndroidSlotsResponse struct {
	Rows []SlotRow `json:"rows"`
}

// ReadAndroidSlots reads all rows from the slot table of the given game.db.
func (s *Service) ReadAndroidSlots(req *ReadAndroidSlotsRequest) (*ReadAndroidSlotsResponse, error) {
	rows, err := ReadSlots(req.DBPath)
	if err != nil {
		return nil, err
	}
	return &ReadAndroidSlotsResponse{Rows: rows}, nil
}

// cachedGameDBDir returns the directory where pulled game.db files are stored.
// It lives in the user cache dir so repeated migrations do not pollute the OS temp.
func cachedGameDBDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "wodima-migrate")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// cachedGameDBPath returns the single destination for Wi-Fi upload.
func cachedGameDBPath() (string, error) {
	dir, err := cachedGameDBDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "game.db"), nil
}

// StartWifiServerRequest carries arguments for StartWifiServer.
type StartWifiServerRequest struct{}

// StartWifiServerResponse is the return envelope for StartWifiServer.
type StartWifiServerResponse struct {
	Result *WifiResult `json:"result"`
}

// StartWifiServer starts an HTTP server on the local network for receiving
// game.db via Wi-Fi upload from the user's phone.
func (s *Service) StartWifiServer(req *StartWifiServerRequest) (*StartWifiServerResponse, error) {
	_ = req
	dst, err := cachedGameDBPath()
	if err != nil {
		return nil, err
	}
	result, err := StartWifiServer(dst)
	if err != nil {
		return nil, err
	}
	return &StartWifiServerResponse{Result: result}, nil
}

// StopWifiServerRequest carries arguments for StopWifiServer.
type StopWifiServerRequest struct{}

// StopWifiServerResponse is the return envelope for StopWifiServer.
type StopWifiServerResponse struct{}

// StopWifiServer stops the currently running Wi-Fi server.
func (s *Service) StopWifiServer(req *StopWifiServerRequest) (*StopWifiServerResponse, error) {
	_ = req
	StopWifiServer()
	return &StopWifiServerResponse{}, nil
}

// CheckWifiUploadRequest carries arguments for CheckWifiUpload.
type CheckWifiUploadRequest struct {
	Token string `json:"token"`
}

// CheckWifiUploadResponse is the return envelope for CheckWifiUpload.
type CheckWifiUploadResponse struct {
	Path string `json:"path"`
}

// CheckWifiUpload polls the Wi-Fi server for upload completion.
// Returns the local file path when upload is done, empty string otherwise.
func (s *Service) CheckWifiUpload(req *CheckWifiUploadRequest) (*CheckWifiUploadResponse, error) {
	path, err := CheckWifiUpload(req.Token)
	if err != nil {
		return nil, err
	}
	return &CheckWifiUploadResponse{Path: path}, nil
}

// PickAndroidDBManuallyRequest carries arguments for PickAndroidDBManually.
type PickAndroidDBManuallyRequest struct{}

// PickAndroidDBManuallyResponse is the return envelope for PickAndroidDBManually.
// Paths contains one or more user-selected game.db files.
type PickAndroidDBManuallyResponse struct {
	Paths []string `json:"paths"`
}

// PickAndroidDBManually opens a file picker allowing the user to choose one or
// more game.db files previously copied to the PC. Multiple selection is supported
// so users who have both a standard install and a TapTap install can pick both
// at once.
func (s *Service) PickAndroidDBManually(req *PickAndroidDBManuallyRequest) (*PickAndroidDBManuallyResponse, error) {
	_ = req
	if s == nil {
		return nil, errors.New("service is not initialized")
	}

	if s.app == nil {
		return nil, errors.New("app is not initialized")
	}

	if s.app.Dialog == nil {
		return nil, errors.New("dialog is not initialized")
	}

	paths, err := s.app.Dialog.OpenFile().
		SetTitle("Select Android game.db (one or more)").
		AddFilter("game.db (*.db, *.sqlite)", "*.db;*.sqlite;game.db").
		AddFilter("All Files", "*.*").
		PromptForMultipleSelection()
	if err != nil {
		return nil, err
	}

	return &PickAndroidDBManuallyResponse{Paths: paths}, nil
}
