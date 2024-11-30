package src

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	up2date "threadfin/src/internal/up2date/client"

	"github.com/hashicorp/go-version"

	"reflect"
)

// BinaryUpdate : Binary Update Prozess. Git Branch master und beta wird von GitHub geladen.
func BinaryUpdate(betaFlag *bool) (err error) {

	if !System.GitHub.Update {
		ShowWarning(2099)
		return
	}

	if !Settings.ThreadfinAutoUpdate {
		ShowWarning(2098)
		return
	}

	var updater = &up2date.Updater
	updater.Name = System.Update.Name
	updater.Branch = System.Branch

	up2date.Init()

	if *betaFlag {
		updater.Branch = "Beta"
	} else {
		updater.Branch = "Main"
	}

	ShowInfo("BRANCH:" + updater.Branch)
	var releaseInfo = fmt.Sprintf("%s/releases", System.Update.Github)
	//var latest string
	//var bin_name string
	var body []byte

	var git []*GithubReleaseInfo

	resp, err := http.Get(releaseInfo)
	if err != nil {
		ShowError(err, 6003)
		return nil
	}

	body, _ = io.ReadAll(resp.Body)

	err = json.Unmarshal(body, &git)
	if err != nil {
		return err
	}

	// Get latest prerelease tag name
	if updater.Branch == "Beta" {
		for _, release := range git {
			if release.Prerelease {
				updater.Response.Version = release.TagName
				updater.Response.UpdatedAt = release.Assets[0].UpdatetAt
				for _, asset := range release.Assets {
					new_asset := up2date.AssetsStruct{DownloadUrl: asset.DownloadUrl, UpdatetAt: asset.UpdatetAt}
					updater.Response.Assets = append(updater.Response.Assets, new_asset)
				}
				break
			}
		}
	}

	// Latest main tag name
	if updater.Branch == "Main" {
		for _, release := range git {
			if !release.Prerelease {
				updater.Response.Version = release.TagName
				for _, asset := range release.Assets {
					new_asset := up2date.AssetsStruct{DownloadUrl: asset.DownloadUrl, UpdatetAt: asset.UpdatetAt}
					updater.Response.Assets = append(updater.Response.Assets, new_asset)
				}
				break
			}
		}
	}

	ShowInfo("TAG LATEST:" + updater.Response.Version)

	for _, asset := range updater.Response.Assets {
		if strings.Contains(asset.DownloadUrl, System.OS) && strings.Contains(asset.DownloadUrl, System.ARCH) {
			updater.Response.Status = true
			updater.Response.UpdateBIN = asset.DownloadUrl
		}
	}

	ShowInfo("FILE:" + updater.Response.UpdateBIN)

	var path_to_file string
	do_upgrade := false
	if updater.Branch == "Beta" {
		if System.Beta {
			existsNewerVersion(updater.Response.Version, updater.Response.Status)
		} else {
			do_upgrade = true
		}
	} else {
		if System.Beta {
			do_upgrade = true
		} else {
			if existsNewerVersion(updater.Response.Version, updater.Response.Status) {
				do_upgrade = true
			}
		}
	}

	// Versionsnummer überprüfen
	if do_upgrade {
		if Settings.ThreadfinAutoUpdate {
			// Update durchführen
			var fileType, url string

			ShowInfo(fmt.Sprintf("Update Available:Version: %s", updater.Response.Version))

			switch System.Branch {

			// Update von GitHub
			case "Master", "Beta":
				ShowInfo("Update Server:GitHub")

			// Update vom eigenen Server
			default:
				ShowInfo(fmt.Sprintf("Update Server:%s", Settings.UpdateURL))

			}

			ShowInfo(fmt.Sprintf("Start Update:Branch: %s", updater.Branch))

			// Neue Version als BIN Datei herunterladen
			if len(updater.Response.UpdateBIN) > 0 {
				url = updater.Response.UpdateBIN
				fileType = "bin"
			}

			// Neue Version als ZIP Datei herunterladen
			if len(updater.Response.UpdateZIP) > 0 {
				url = updater.Response.UpdateZIP
				fileType = "zip"
			}

			if len(url) > 0 {

				err = up2date.DoUpdate(fileType, updater.Response.Filename)
				if err != nil {
					ShowError(err, 6002)
				}
				if System.Branch == "Beta" {
					if err := os.WriteFile(path_to_file, []byte(updater.Response.UpdatedAt), 0666); err != nil {
						ShowError(err, 6005)
					}
				}
			}

		} else {
			// Hinweis ausgeben
			ShowWarning(6004)
		}

	} else {
		ShowInfo("BIN:Update omitted")
	}

	return nil
}

func existsNewerVersion(downloadableVersion string, status bool) bool {
	var currentVersion = System.Version + "." + System.Build
	current_version, _ := version.NewVersion(currentVersion)
	response_version, _ := version.NewVersion(downloadableVersion)
	if response_version.GreaterThan(current_version) && status {
		return true
	}
	return false
}

func conditionalUpdateChanges() (err error) {

checkVersion:
	settingsMap, err := loadJSONFileToMap(System.File.Settings)
	if err != nil || len(settingsMap) == 0 {
		return
	}

	if settingsVersion, ok := settingsMap["version"].(string); ok {

		if settingsVersion > System.DBVersion {
			ShowInfo("Settings DB Version:" + settingsVersion)
			ShowInfo("System DB Version:" + System.DBVersion)
			err = errors.New(getErrMsg(1031))
			return
		}

		// Letzte Kompatible Version (1.4.4)
		if settingsVersion < System.Compatibility {
			err = errors.New(getErrMsg(1013))
			return
		}

		switch settingsVersion {

		case "1.4.4":
			// UUID Wert in xepg.json setzen
			err = setValueForUUID()
			if err != nil {
				return
			}

			// Neuer Filter (WebUI). Alte Filtereinstellungen werden konvertiert
			if oldFilter, ok := settingsMap["filter"].([]interface{}); ok {
				var newFilterMap = convertToNewFilter(oldFilter)
				settingsMap["filter"] = newFilterMap

				settingsMap["version"] = "2.0.0"

				err = saveMapToJSONFile(System.File.Settings, settingsMap)
				if err != nil {
					return
				}

				goto checkVersion

			} else {
				err = errors.New(getErrMsg(1030))
				return
			}

		case "2.0.0":

			if oldBuffer, ok := settingsMap["buffer"].(bool); ok {

				var newBuffer string
				switch oldBuffer {
				case true:
					newBuffer = "threadfin"
				case false:
					newBuffer = "-"
				}

				settingsMap["buffer"] = newBuffer

				settingsMap["version"] = "2.1.0"

				err = saveMapToJSONFile(System.File.Settings, settingsMap)
				if err != nil {
					return
				}

				goto checkVersion

			} else {
				err = errors.New(getErrMsg(1030))
				return
			}

		case "2.1.0":
			// Falls es in einem späteren Update Änderungen an der Datenbank gibt, geht es hier weiter

			break
		}

	} else {
		// settings.json ist zu alt (älter als Version 1.4.4)
		err = errors.New(getErrMsg(1013))
	}

	return
}

func convertToNewFilter(oldFilter []interface{}) (newFilterMap map[int]interface{}) {

	newFilterMap = make(map[int]interface{})

	switch reflect.TypeOf(oldFilter).Kind() {

	case reflect.Slice:
		s := reflect.ValueOf(oldFilter)

		for i := 0; i < s.Len(); i++ {

			var newFilter FilterStruct
			newFilter.Active = true
			newFilter.Name = fmt.Sprintf("Custom filter %d", i+1)
			newFilter.Filter = s.Index(i).Interface().(string)
			newFilter.Type = "custom-filter"
			newFilter.CaseSensitive = false

			newFilterMap[i] = newFilter

		}

	}

	return
}

func setValueForUUID() (err error) {

	xepg, err := loadJSONFileToMap(System.File.XEPG)
	if err == nil {

		for _, c := range xepg {

			var xepgChannel = c.(map[string]interface{})

			if uuidKey, ok := xepgChannel["_uuid.key"].(string); ok {

				if value, ok := xepgChannel[uuidKey].(string); ok {

					if len(value) > 0 {
						xepgChannel["_uuid.value"] = value
					}

				}

			}

		}
	}
	err = saveMapToJSONFile(System.File.XEPG, xepg)

	return
}
