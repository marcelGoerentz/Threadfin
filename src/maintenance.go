package src

import (
	"fmt"
	"math/rand"
	"time"
)

// InitMaintenance : Wartungsprozess initialisieren
func InitMaintenance() (err error) {

	System.TimeForAutoUpdate = fmt.Sprintf("0%d%d", randomTime(0, 2), randomTime(10, 59))

	go maintenance()

	return
}

func maintenance() {

	for {

		var t = time.Now()

		// Aktualisierung der Playlist und XMLTV Dateien
		if System.ScanInProgress == 0 {

			for _, schedule := range Settings.Update {

				if schedule == t.Format("1504") {

					ShowInfo("Update:" + schedule)

					// Backup erstellen
					err := ThreadfinAutoBackup()
					if err != nil {
						ShowError(err, 000)
					}

					// Playlist und XMLTV Dateien aktualisieren
					getProviderData("m3u", "")
					getProviderData("hdhr", "")

					if Settings.EpgSource == "XEPG" {
						getProviderData("xmltv", "")
					}

					// Datenbank für DVR erstellen
					err = buildDatabaseDVR()
					if err != nil {
						ShowError(err, 000)
					}

					if !Settings.CacheImages && System.ImageCachingInProgress == 0 {
						removeChildItems(System.Folder.ImagesCache)
					}

					// XEPG Dateien erstellen
					Data.Cache.XMLTV = make(map[string]XMLTV)
					buildXEPG(false)

				}

			}

			// Update Threadfin (Binary)
			if System.TimeForAutoUpdate == t.Format("1504") {
				BinaryUpdate()
			}

		}

		time.Sleep(60 * time.Second)

	}
}

func randomTime(min, max int) int {
	rand.NewSource(time.Now().Unix())
	return rand.Intn(max-min) + min
}
