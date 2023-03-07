package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/axllent/semver"
	"github.com/axllent/ssbak/app"
)

// AllowPrereleases defines whether pre-releases may be included
var AllowPrereleases = false

// Releases struct for Github releases json
type Releases []struct {
	Name       string `json:"name"`       // release name
	Tag        string `json:"tag_name"`   // release tag
	Prerelease bool   `json:"prerelease"` // Github pre-release
	Assets     []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
		ID                 int64  `json:"id"`
		Name               string `json:"name"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// Release struct contains the file data for downloadable release
type Release struct {
	Name string
	Tag  string
	URL  string
	Size int64
}

// GithubLatest fetches the latest release info & returns release tag, filename & download url
func GithubLatest(repo, name string) (string, string, string, error) {
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)

	app.Log(fmt.Sprintf("Downloading releases from %s", releaseURL))

	resp, err := http.Get(releaseURL) // #nosec
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", "", "", err
	}

	linkOS := runtime.GOOS
	linkArch := runtime.GOARCH
	linkExt := ".tar.gz"
	if linkOS == "windows" {
		// Windows uses .zip instead
		linkExt = ".zip"
	}

	var allReleases = []Release{}

	var releases Releases

	if err := json.Unmarshal(body, &releases); err != nil {
		return "", "", "", err
	}

	archiveName := fmt.Sprintf("%s_%s_%s%s", name, linkOS, linkArch, linkExt)

	// loop through releases
	for _, r := range releases {
		if !semver.IsValid(r.Tag) {
			// Invalid semversion, skip
			continue
		}

		if !AllowPrereleases && (semver.Prerelease(r.Tag) != "" || r.Prerelease) {
			// we don't accept AllowPrereleases, skip
			continue
		}

		for _, a := range r.Assets {
			if a.Name == archiveName {
				thisRelease := Release{a.Name, r.Tag, a.BrowserDownloadURL, a.Size}
				allReleases = append(allReleases, thisRelease)
				break
			}
		}
	}

	if len(allReleases) == 0 {
		// no releases with suitable assets found
		return "", "", "", fmt.Errorf("No binary releases found")
	}

	var latestRelease = Release{}

	for _, r := range allReleases {
		// detect the latest release
		if semver.Compare(r.Tag, latestRelease.Tag) == 1 {
			latestRelease = r
		}
	}

	return latestRelease.Tag, latestRelease.Name, latestRelease.URL, nil
}

// GreaterThan compares the current version to a different version
// returning < 1 not upgradeable
func GreaterThan(toVer, fromVer string) bool {
	return semver.Compare(toVer, fromVer) == 1
}

// GithubUpdate the running binary with the latest release binary from Github
func GithubUpdate(repo, appName, currentVersion string) (string, error) {
	ver, filename, downloadURL, err := GithubLatest(repo, appName)

	if err != nil {
		return "", err
	}

	if ver == currentVersion {
		return "", fmt.Errorf("No new release found")
	}

	if semver.Compare(ver, currentVersion) < 1 {
		return "", fmt.Errorf("No newer releases found (latest %s)", ver)
	}

	tmpDir := app.GetTempDir()

	// outFile can be a tar.gz or a zip, depending on architecture
	outFile := filepath.Join(tmpDir, filename)

	if err := DownloadToFile(downloadURL, outFile); err != nil {
		return "", err
	}

	newExec := filepath.Join(tmpDir, "ssbak")

	app.Log(fmt.Sprintf("Extracting %s", outFile))

	if runtime.GOOS == "windows" {
		if _, err := Unzip(outFile, tmpDir); err != nil {
			return "", err
		}
		newExec = filepath.Join(tmpDir, "ssbak.exe")
	} else {
		if err := TarGZExtract(outFile, tmpDir); err != nil {
			return "", err
		}
	}

	app.AddTempFile(outFile)
	app.AddTempFile(newExec)

	// get the running binary
	oldExec, err := os.Executable()
	if err != nil {
		panic(err)
	}

	app.Log(fmt.Sprintf("Replacing %s with %s", oldExec, newExec))

	if err = ReplaceFile(oldExec, newExec); err != nil {
		return "", err
	}

	return ver, nil
}

// DownloadToFile downloads a URL to a file
func DownloadToFile(url, filepath string) error {
	app.Log(fmt.Sprintf("Downloading %s to %s", url, filepath))

	// Get the data
	resp, err := http.Get(url) // #nosec
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(path.Clean(filepath))
	if err != nil {
		return err
	}

	defer func() {
		if err := out.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)

	return err
}

// ReplaceFile replaces one file with another.
// Running files cannot be overwritten, so it has to be moved
// and the new binary saved to the original path. This requires
// read & write permissions to both the original file and directory.
// Note, on Windows it is not possible to delete a running program,
// so the old exe is renamed and moved to os.TempDir()
func ReplaceFile(dst, src string) error {
	// open the source file for reading
	source, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}

	// destination directory eg: /usr/local/bin
	dstDir := filepath.Dir(dst)
	// binary filename
	binaryFilename := filepath.Base(dst)
	// old binary tmp name
	dstOld := fmt.Sprintf("%s.old", binaryFilename)
	// new binary tmp name
	dstNew := fmt.Sprintf("%s.new", binaryFilename)
	// absolute path of new tmp file
	newTmpAbs := filepath.Join(dstDir, dstNew)
	// absolute path of old tmp file
	oldTmpAbs := filepath.Join(dstDir, dstOld)

	// get src permissions
	fi, _ := os.Stat(dst)
	srcPerms := fi.Mode().Perm()

	// create the new file
	tmpNew, err := os.OpenFile(filepath.Clean(newTmpAbs), os.O_CREATE|os.O_RDWR, srcPerms) // #nosec
	if err != nil {
		return err
	}

	// copy new binary to <binary>.new
	if _, err := io.Copy(tmpNew, source); err != nil {
		return err
	}

	// close immediately else Windows has a fit
	if err := tmpNew.Close(); err != nil {
		return err
	}

	if err := source.Close(); err != nil {
		return err
	}

	// rename the current executable to <binary>.old
	if err := os.Rename(dst, oldTmpAbs); err != nil {
		return err
	}

	// rename the <binary>.new to current executable
	if err := os.Rename(newTmpAbs, dst); err != nil {
		return err
	}

	// delete the old binary
	if runtime.GOOS == "windows" {
		tmpDir := os.TempDir()
		delFile := filepath.Join(tmpDir, filepath.Base(oldTmpAbs))
		if err := os.Rename(oldTmpAbs, delFile); err != nil {
			return err
		}
	} else {
		if err := os.Remove(oldTmpAbs); err != nil {
			return err
		}
	}

	// remove the src file
	if err := os.Remove(src); err != nil {
		return err
	}

	return nil
}
