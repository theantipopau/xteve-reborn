package up2date

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// downloadClient bounds how long an update download may take. Release zips
// are ~10 MB; the overall timeout is generous for slow links but still
// guarantees a dead mirror can't hang the updater forever.
var downloadClient = &http.Client{
	Timeout: 10 * time.Minute,
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 15 * time.Second}).DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	},
}

const oldBinaryPrefix = "_old_"

// DoUpdate downloads the release zip at zipURL, verifies it against the
// SHA-256 listed for it in the release's checksums file, extracts
// binaryName from it, swaps it in for the running executable and restarts.
// The running binary is kept as _old_<name> and restored if any step fails.
func DoUpdate(zipURL, checksumsURL, binaryName string) (err error) {

	if len(zipURL) == 0 {
		return errors.New("no download URL for this platform")
	}

	if len(checksumsURL) == 0 {
		return errors.New("release has no checksums file; refusing to install an unverified binary")
	}

	if runtime.GOOS == "windows" {
		binaryName = binaryName + ".exe"
	}

	binary, err := os.Executable()
	if err != nil {
		return err
	}

	binary, err = filepath.EvalSymlinks(binary)
	if err != nil {
		return err
	}

	var dir = filepath.Dir(binary)

	// Everything is staged next to the binary so the final rename stays on
	// one filesystem (a cross-device rename would fail).
	tmpZip, err := os.CreateTemp(dir, ".xteve-reborn-update-*.zip")
	if err != nil {
		return fmt.Errorf("can't write to %s: %w", dir, err)
	}
	defer os.Remove(tmpZip.Name())

	log.Println("[UPDATE]", "Downloading", zipURL)

	var hash = sha256.New()
	err = download(zipURL, io.MultiWriter(tmpZip, hash))
	tmpZip.Close()
	if err != nil {
		return err
	}

	expected, err := expectedChecksum(checksumsURL, filepath.Base(zipURL))
	if err != nil {
		return err
	}

	if actual := hex.EncodeToString(hash.Sum(nil)); strings.EqualFold(actual, expected) == false {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", filepath.Base(zipURL), actual, expected)
	}

	log.Println("[UPDATE]", "Checksum verified")

	tmpBinary, err := os.CreateTemp(dir, ".xteve-reborn-update-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpBinary.Name())

	err = extractFile(tmpZip.Name(), binaryName, tmpBinary)
	tmpBinary.Close()
	if err != nil {
		return err
	}

	if err = os.Chmod(tmpBinary.Name(), 0755); err != nil {
		return err
	}

	// Swap: running binary -> _old_<name>, new binary -> original path.
	// Renaming a running executable is allowed on Windows as well as Unix;
	// deleting it isn't on Windows, which is why CleanupOldBinary exists.
	var oldBinary = filepath.Join(dir, oldBinaryPrefix+filepath.Base(binary))
	os.Remove(oldBinary)

	if err = os.Rename(binary, oldBinary); err != nil {
		return err
	}

	if err = os.Rename(tmpBinary.Name(), binary); err != nil {
		os.Rename(oldBinary, binary)
		return err
	}

	log.Println("[UPDATE]", "Update installed, restarting")

	if runtime.GOOS == "windows" {

		var cmd = exec.Command(binary, os.Args[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

		if err = cmd.Start(); err != nil {
			os.Remove(binary)
			os.Rename(oldBinary, binary)
			return err
		}

		os.Exit(0)
	}

	err = syscall.Exec(binary, os.Args, os.Environ())

	// Only reached if exec failed - the old process is still running, so
	// put its binary back.
	os.Remove(binary)
	os.Rename(oldBinary, binary)

	return err
}

// CleanupOldBinary removes the _old_<name> backup left by a previous update.
// On Windows the running executable can't be deleted, so the old binary
// survives the restart and is removed on the next start instead.
func CleanupOldBinary() {

	binary, err := os.Executable()
	if err != nil {
		return
	}

	binary, err = filepath.EvalSymlinks(binary)
	if err != nil {
		return
	}

	os.Remove(filepath.Join(filepath.Dir(binary), oldBinaryPrefix+filepath.Base(binary)))
}

func download(url string, w io.Writer) (err error) {

	resp, err := downloadClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

// expectedChecksum downloads a sha256sum-format checksums file
// ("<hex>  <filename>" per line) and returns the hash listed for filename.
func expectedChecksum(checksumsURL, filename string) (string, error) {

	var sb strings.Builder
	if err := download(checksumsURL, &sb); err != nil {
		return "", err
	}

	return parseChecksum(sb.String(), filename)
}

func parseChecksum(checksums, filename string) (string, error) {

	var scanner = bufio.NewScanner(strings.NewReader(checksums))

	for scanner.Scan() {

		var fields = strings.Fields(scanner.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == filename {
			return fields[0], nil
		}

	}

	return "", fmt.Errorf("no checksum listed for %s", filename)
}

// extractFile copies the single zip entry named name into w. Only an exact
// top-level match is accepted, so a crafted archive can't write outside the
// staging file (the previous extractor joined entry paths onto a directory
// unchecked, i.e. "zip slip").
func extractFile(archive, name string, w io.Writer) (err error) {

	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {

		if file.Name != name {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		_, err = io.Copy(w, rc)
		return err
	}

	return fmt.Errorf("%s not found in update archive", name)
}
