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
	"time"

	"github.com/hashicorp/go-version"

	"reflect"
)

// BinaryUpdate : Binary Update Prozess. Git Branch master und beta wird von GitHub geladen.
func BinaryUpdate() (err error) {

	if !System.GitHub.Update {
		showWarning(2099)
		return
	}

	if !Settings.ThreadfinAutoUpdate {
		showWarning(2098)
		return
	}

	var debug string

	var updater = &up2date.Updater
	updater.Name = System.Update.Name
	updater.Branch = System.Branch

	up2date.Init()

	showInfo("BRANCH:" + System.Branch)
	switch System.Branch {

	// Update von GitHub
	case "Main", "Beta":
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
		if System.Branch == "Beta" {
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
		if System.Branch == "Main" {
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

		showInfo("TAG LATEST:" + updater.Response.Version)

		for _, asset := range updater.Response.Assets {
			if strings.Contains(asset.DownloadUrl, System.OS) && strings.Contains(asset.DownloadUrl, System.ARCH) {
				updater.Response.Status = true
				updater.Response.UpdateBIN = asset.DownloadUrl
			}
		}

		showInfo("FILE:" + updater.Response.UpdateBIN)

	// Update vom eigenen Server
	default:

		updater.URL = Settings.UpdateURL

		if len(updater.URL) == 0 {
			showInfo(fmt.Sprintf("Update URL:No server URL specified, update will not be performed. Branch: %s", System.Branch))
			return
		}

		showInfo("Update URL:" + updater.URL)
		fmt.Println("-----------------")

		// Versionsinformationen vom Server laden
		err = up2date.GetVersion()
		if err != nil {

			debug = fmt.Sprint(err.Error())
			showDebug(debug, 1)

			return nil
		}

		if len(updater.Response.Reason) > 0 {

			err = fmt.Errorf(fmt.Sprintf("Update Server: %s", updater.Response.Reason))
			ShowError(err, 6002)

			return nil
		}

	}

	var path_to_file string
	do_upgrade := false
	if System.Branch == "Beta" {
		path_to_file = System.Folder.Config + "latest_beta_update"
		// If update file does not exits then update the binary to make sure that the latest version is installed
		if _, err := os.Stat(path_to_file); errors.Is(err, os.ErrNotExist) {
			do_upgrade = true
		} else {
			// If the file exists check if the latest-release is newer then the last update
			saved_last_update_date, err := os.ReadFile(path_to_file)
			if err != nil {
				ShowError(err, 0)
				do_upgrade = true
			}
			last_time_date, _ := time.Parse(time.RFC3339, string(saved_last_update_date))
			latest_beta_date, _ := time.Parse(time.RFC3339, updater.Response.UpdatedAt)

			if last_time_date.Before(latest_beta_date) {
				do_upgrade = true
			}
		}
	} else {
		var currentVersion = System.Version + "." + System.Build
		current_version, _ := version.NewVersion(currentVersion)
		response_version, _ := version.NewVersion(updater.Response.Version)
		if response_version.GreaterThan(current_version) && updater.Response.Status {
			do_upgrade = true
		}
	}

	// Versionsnummer überprüfen
	if do_upgrade {
		if Settings.ThreadfinAutoUpdate {
			// Update durchführen
			var fileType, url string

			showInfo(fmt.Sprintf("Update Available:Version: %s", updater.Response.Version))

			switch System.Branch {

			// Update von GitHub
			case "Master", "Beta":
				showInfo("Update Server:GitHub")

			// Update vom eigenen Server
			default:
				showInfo(fmt.Sprintf("Update Server:%s", Settings.UpdateURL))

			}

			showInfo(fmt.Sprintf("Start Update:Branch: %s", updater.Branch))

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
			showWarning(6004)
		}

	} else {
		showInfo("BIN:Update omitted")
	}

	return nil
}

func conditionalUpdateChanges() (err error) {

checkVersion:
	settingsMap, err := loadJSONFileToMap(System.File.Settings)
	if err != nil || len(settingsMap) == 0 {
		return
	}

	if settingsVersion, ok := settingsMap["version"].(string); ok {

		if settingsVersion > System.DBVersion {
			showInfo("Settings DB Version:" + settingsVersion)
			showInfo("System DB Version:" + System.DBVersion)
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
