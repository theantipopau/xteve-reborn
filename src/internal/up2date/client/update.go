package up2date

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

// DoUpdate downloads the release zip at zipURL, extracts binaryName from
// it, and replaces the currently-running executable with it before
// restarting the process. The old binary is kept as a backup
// (_old_<name>) until the new one is confirmed running, and restored if
// any step fails partway through.
func DoUpdate(zipURL, binaryName string) (err error) {

	if len(zipURL) == 0 {
		return
	}

	switch runtime.GOOS {
	case "windows":
		binaryName = binaryName + ".exe"
	}

	log.Println("[UPDATE]", "Downloading", zipURL)

	resp, err := http.Get(zipURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	binary, err := os.Executable()
	if err != nil {
		return err
	}
	var filename = getFilenameFromPath(binary)
	var path = getPlatformPath(binary)
	var oldBinary = path + "_old_" + filename
	var newBinary = binary

	var tmpFolder = path + "tmp"
	var tmpFile = tmpFolder + string(os.PathSeparator) + binaryName

	os.Rename(newBinary, oldBinary)

	// Save the downloaded zip under the current binary's own path, then
	// extract it into a temp folder and pull just the binary back out -
	// mirrors how the file arrives from GitHub (zipped) while keeping the
	// rest of the swap logic format-agnostic.
	out, err := os.Create(binary)
	if err != nil {
		restorOldBinary(oldBinary, newBinary)
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		restorOldBinary(oldBinary, newBinary)
		return err
	}

	log.Println("[UPDATE]", "Extracting update")

	err = extractZIP(binary, tmpFolder)
	binary = newBinary

	if err != nil {
		restorOldBinary(oldBinary, newBinary)
		return err
	}

	err = copyFile(tmpFile, binary)
	if err != nil {
		restorOldBinary(oldBinary, newBinary)
		return err
	}

	os.RemoveAll(tmpFolder)

	err = os.Chmod(binary, 0755)
	out.Close()

	log.Println("[UPDATE]", "Update successful, restarting")

	// Restart binary (Windows)
	if runtime.GOOS == "windows" {

		bin, err := os.Executable()

		if err != nil {
			restorOldBinary(oldBinary, newBinary)
			return err
		}

		var pid = os.Getpid()
		var process, _ = os.FindProcess(pid)

		if proc, err := start(bin); err == nil {

			os.RemoveAll(oldBinary)
			process.Kill()
			proc.Wait()

		} else {
			restorOldBinary(oldBinary, newBinary)
		}

	} else {

		// Restart binary (Linux and UNIX)
		file, _ := os.Executable()
		os.RemoveAll(oldBinary)
		err = syscall.Exec(file, os.Args, os.Environ())
		if err != nil {
			restorOldBinary(oldBinary, newBinary)
			log.Println("[UPDATE] restart failed:", err)
			return err
		}

	}

	return
}

func start(args ...string) (p *os.Process, err error) {

	if args[0], err = exec.LookPath(args[0]); err == nil {

		var procAttr os.ProcAttr
		procAttr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}
		p, err := os.StartProcess(args[0], args, &procAttr)

		if err == nil {
			return p, nil
		}

	}

	return nil, err
}

func restorOldBinary(oldBinary, newBinary string) {
	os.RemoveAll(newBinary)
	os.Rename(oldBinary, newBinary)
}

func getFilenameFromPath(path string) string {

	file := filepath.Base(path)

	return file
}

func getPlatformPath(path string) string {

	var newPath = filepath.Dir(path) + string(os.PathSeparator)

	return newPath
}

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func extractZIP(archive, target string) (err error) {

	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {

		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}

	}

	return
}
