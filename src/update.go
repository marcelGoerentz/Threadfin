package src

import (
	"fmt"
	updater "threadfin/src/internal/updater"
)

// BinaryUpdate : Binary Update Prozess. Git Branch master und beta wird von GitHub geladen.
func BinaryUpdate(forceUpdate bool) bool {

	if !System.GitHub.Update {
		ShowWarning(2099)
		return false
	}

	if !forceUpdate {
		if !Settings.ThreadfinAutoUpdate {
			ShowWarning(2098)
			return false
		}
	}

	var updater = updater.Init(System.Branch, System.Update.Name, System.Update.Git)

	if System.Beta {
		updater.Branch = "beta"
	} else {
		updater.Branch = "master"
	}

	ShowInfo("Update Version:" + updater.Branch)
	var releaseInfoURL = fmt.Sprintf("%s/releases", System.Update.Github)

	if updater.GetBinaryDownloadURL(releaseInfoURL) != nil {
		return false
	}

	if !updater.Response.Status {
		return false
	}

	ShowInfo("LATEST VERSION:" + updater.Response.Version)
	ShowInfo("FILE:" + updater.BinaryDownloadURL)

	do_upgrade := false
	if !forceUpdate {
		do_upgrade = updater.ExistsNewerVersion(System.Version, System.Build)
	} else {
		do_upgrade = true
	}

	// Versionsnummer überprüfen
	if do_upgrade {
		err := updater.DoUpdateNew()
		if err != nil {
			ShowError(err, 6002)
			return false
		}
		return true
	} else {
		ShowInfo("BIN:Update omitted")
		return false
	}
}
