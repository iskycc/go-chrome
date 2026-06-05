package browser

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const cftJSON = "https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json"

// DownloadProgress is called during download.
type DownloadProgress func(downloaded, total int64)

// VersionManifest records installed Chrome info.
type VersionManifest struct {
	Version      string    `json:"version"`
	Source       string    `json:"source"` // official, custom_url
	Channel      string    `json:"channel,omitempty"`
	DownloadURL  string    `json:"downloadURL,omitempty"`
	CustomLabel  string    `json:"customLabel,omitempty"`
	InstalledAt  time.Time `json:"installedAt"`
	ExePath      string    `json:"exePath"`
}

// CFTVersions represents the JSON structure from Chrome for Testing.
type CFTVersions struct {
	Timestamp string                `json:"timestamp"`
	Versions  map[string]CFTChannel `json:"versions"`
}

// CFTChannel holds per-channel data.
type CFTChannel struct {
	Version  string              `json:"version"`
	Revision string              `json:"revision"`
	Downloads map[string][]CFTDL `json:"downloads"`
}

// CFTDL holds a single download entry.
type CFTDL struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
}

// FetchStableInfo fetches the latest stable Chrome for Testing info.
func FetchStableInfo() (version, url, hash string, err error) {
	resp, err := http.Get(cftJSON)
	if err != nil {
		return "", "", "", fmt.Errorf("fetch versions: %w", err)
	}
	defer resp.Body.Close()
	var data CFTVersions
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", "", fmt.Errorf("decode versions: %w", err)
	}
	ch, ok := data.Versions["Stable"]
	if !ok {
		return "", "", "", fmt.Errorf("stable channel not found")
	}
	version = ch.Version
	for _, dl := range ch.Downloads["chrome"] {
		if dl.Platform == "win64" || dl.Platform == "win32" {
			url = dl.URL
			hash = dl.SHA256
			break
		}
	}
	if url == "" {
		return "", "", "", fmt.Errorf("no win64/win32 download found")
	}
	return version, url, hash, nil
}

// DownloadFile downloads a file with progress callback.
func DownloadFile(url, dest string, progress DownloadProgress) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	var written int64
	total := resp.ContentLength
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if progress != nil {
				progress(written, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// VerifySHA256 checks file hash.
func VerifySHA256(path, expected string) error {
	if expected == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", got, expected)
	}
	return nil
}

// ExtractZIP extracts a zip file to a destination directory.
func ExtractZIP(src, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// FindChromeExe recursively searches for chrome.exe under dir.
func FindChromeExe(dir string) (string, error) {
	var found string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // continue
		}
		if info.IsDir() {
			return nil
		}
		if strings.EqualFold(info.Name(), "chrome.exe") {
			found = path
			return io.EOF // stop walking
		}
		return nil
	})
	if found != "" {
		return found, nil
	}
	if err != nil && err != io.EOF {
		return "", err
	}
	return "", fmt.Errorf("chrome.exe not found in %s", dir)
}
