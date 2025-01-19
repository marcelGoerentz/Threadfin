package up2date

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
)

// ClientInfo : Information about the key (NAME OS, ARCH, UUID, KEY)
type ClientInfo struct {
	Arch   string `json:"arch"`
	Branch string `json:"branch"`
	CMD    string `json:"cmd,omitempty"`
	Name   string `json:"name"`
	OS     string `json:"os"`
	URL    string `json:"url"`

	Filename string `json:"filename,omitempty"`
	BinaryDownloadURL string `json:"binaryDownloadURL,omitempty"`
	SHA256DownloadURL string `json:"sha256DownloadURL,omitempty"`

	Response ServerResponse `json:"response,omitempty"`

	Server *http.Server
}

// ServerResponse : Response from server after client request
type ServerResponse struct {
	Status    bool   `json:"status,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Version   string `json:"version,omitempty"`
	UpdateBIN string `json:"update.url.bin,omitempty"`
	UpdateZIP string `json:"update.url.zip,omitempty"`
	UpdatedAt string
	Assets    []AssetsStruct
}

type GithubReleaseInfo struct {
	TagName    string         `json:"tag_name"`
	Prerelease bool           `json:"prerelease"`
	Assets     []AssetsStruct `json:"assets"`
}

type AssetsStruct struct {
	Name        string `json:"name"`
	DownloadUrl string `json:"browser_download_url"`
	UpdatetAt   string `json:"updated_at"`
	ContentType string `json:"content_type"`
}

// Updater : Client infos
var Updater ClientInfo

// UpdateURL : URL for the new binary
var UpdateURL string

// Init : Init
func Init(branch, name, url string) *ClientInfo {
	return &ClientInfo{
		Arch:   runtime.GOARCH,
		Branch: branch,
		Name:   name,
		OS:     runtime.GOOS,
		URL:    url,
	}
}

func (c *ClientInfo) GetBinaryDownloadURL(releasesURL string) error {
	//var latest string
	//var bin_name string
	var body []byte

	var git []*GithubReleaseInfo

	resp, err := http.Get(releasesURL)
	if err != nil {
		return err
	}

	body, _ = io.ReadAll(resp.Body)

	err = json.Unmarshal(body, &git)
	if err != nil {
		return err
	}

	// Get latest prerelease tag name
	if c.Branch == "beta" {
		for _, release := range git {
			if release.Prerelease {
				c.Response.Version = release.TagName
				c.Response.Assets = append(c.Response.Assets, release.Assets...)
			}
		}
	}

	// Latest main tag name
	if c.Branch == "master" {
		for _, release := range git {
			if !release.Prerelease {
				c.Response.Version = release.TagName
				c.Response.Assets = append(c.Response.Assets, release.Assets...)
			}
		}
	}

	// Find the download link corresponding to the OS
	foundBinaryURL := false
	foundSHA256URL := false
	for _, asset := range c.Response.Assets {
		if strings.Contains(asset.Name, c.OS) && strings.Contains(asset.Name, c.Arch) {
			if strings.Contains(asset.Name, "sha256") {
				c.SHA256DownloadURL = asset.DownloadUrl
				foundSHA256URL = true
			} else {
				c.BinaryDownloadURL = asset.DownloadUrl
				foundBinaryURL = true
			}
			foundSHA256URL = true
			if foundBinaryURL && foundSHA256URL {
				c.Filename = asset.Name
				c.Response.Status = true
				break
			}
		}
	}
	return nil
}

func (c *ClientInfo) ExistsNewerVersion(Version, Build string) bool {
	var currentVersion = Version + "." + Build
	current_version, _ := version.NewVersion(currentVersion)
	response_version, _ := version.NewVersion(c.Response.Version)
	if response_version == nil {
		current_date := getBinaryTime()
		layout := time.RFC3339
		response_date, err := time.Parse(layout, c.Response.UpdatedAt)
		if err != nil {
			return false
		}
		if current_date.Before(response_date) {
			return true
		}
	} else if response_version.GreaterThan(current_version) && c.Response.Status {
		return true
	}
	return false
}

func getBinaryTime() time.Time {
	executablePath, err := os.Executable()
	if err != nil {
		return time.Now()
	}
	binaryInfo, err := os.Stat(executablePath)
	if err != nil {
		return time.Now()
	}
	return binaryInfo.ModTime()
}

func (c *ClientInfo) DoUpdateNew() error {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	resp, err := client.Get(c.BinaryDownloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create a temporary file to save the new binary
	tmpBinary, err := os.CreateTemp("", c.Filename)
	if err != nil {
		return err
	}

	buf := make ([]byte, 128 * 1024)
	for {
		n , err := resp.Body.Read(buf)
		if err != nil  && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := tmpBinary.Write(buf[:n]); err != nil {
			return err
		}
	}

	// Close the file explicitly
	tmpBinary.Close()

	if err := c.verifyChecksum(tmpBinary.Name()); err != nil {
		os.Remove(tmpBinary.Name())
		return err
	}

	// Get the current executable
	exePath, err := os.Executable()
	if err != nil {
		os.Remove(tmpBinary.Name())
		return err
	}

	// Get the stat from the current executable
    oldExeInfo, err := os.Stat(exePath)
    if err != nil {
		os.Remove(tmpBinary.Name())
        return err
    }

    // Set attributes of the new executable
    if err := os.Chmod(tmpBinary.Name(), oldExeInfo.Mode()); err != nil {
		os.Remove(tmpBinary.Name())
        return err
    }

	// Backup the old executable
	backupPath := exePath + ".old"
	_ = os.Remove(backupPath)

	// Move the existing executable
	err = os.Rename(exePath, backupPath)
	if err != nil {
		os.Remove(tmpBinary.Name())
		return err
	}

	// Move the new executable
	if err := os.Rename(tmpBinary.Name(), exePath); err != nil {
		copyFile(backupPath, exePath)
		return err
	}

	// Stop the webserver
	if err = shutdownWebserver(c.Server); err != nil {
		return err
	}

	// Restart the application
	restartApplication(exePath, os.Args, os.Environ())

	return nil
}

func shutdownWebserver(server *http.Server) error {
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		err := server.Shutdown(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func restartApplication(exePath string, args []string, env []string) error {
	

    cmd := exec.Command(exePath, args[1:]...) // Pass all arguments except the first one (which is the executable path)
    cmd.Env = env
    err := cmd.Start()
    if err != nil {
        return err
	}

	if runtime.GOOS == "windows" {

		var pid = os.Getpid()
		var process, _ = os.FindProcess(pid)

		process.Kill() // Kill this application
		process.Wait()

	} else {
		os.Exit(0) // Stop this application 
	}
    return nil
}

func copyFile(src, dst string) error {
    sourceFile, err := os.Open(src)
    if err != nil {
        return err
    }
    defer sourceFile.Close()

    destinationFile, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer destinationFile.Close()

    _, err = io.Copy(destinationFile, sourceFile)
    return err
}

func (c *ClientInfo) verifyChecksum(filePath string) error {
	// Download the expected checksum
	if c.SHA256DownloadURL == "" {
		return nil
	}
	resp, err := http.Get(c.SHA256DownloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var expectedChecksum string
	if _, err := fmt.Fscan(resp.Body, &expectedChecksum); err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	calculatedChecksum := hex.EncodeToString(hash.Sum(nil))
	
	if calculatedChecksum != expectedChecksum {
		return fmt.Errorf("checksum verification failed: expected %s, got %s", expectedChecksum, calculatedChecksum)
	}
	return nil
}