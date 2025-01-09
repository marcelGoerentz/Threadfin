package src

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
)

var onlyOnce = false

// Entwicklerinfos anzeigen
func showDevInfo() {

	if System.Dev {

		fmt.Print("\033[31m")
		fmt.Println("* * * * * D E V   M O D E * * * * *")
		fmt.Println("Version: ", System.Version)
		fmt.Println("Build:   ", System.Build)
		fmt.Println("* * * * * * * * * * * * * * * * * *")
		fmt.Print("\033[0m")
		fmt.Println()

	}
}

// Alle Systemordner erstellen
func createSystemFolders() (err error) {

	e := reflect.ValueOf(&System.Folder).Elem()

	for i := 0; i < e.NumField(); i++ {

		var folder = e.Field(i).Interface().(string)

		err = checkFolder(folder)

		if err != nil {
			return
		}

	}

	return
}

// Alle Systemdateien erstellen
func createSystemFiles() (err error) {
	var debug string
	for _, file := range SystemFiles {

		var filename = getPlatformFile(System.Folder.Config + file)

		err = checkFile(filename)
		if err != nil {
			// File does not exist, will be created now
			err = saveMapToJSONFile(filename, make(map[string]interface{}))
			if err != nil {
				return
			}

			debug = fmt.Sprintf("Create File:%s", filename)
			ShowDebug(debug, 1)

		}

		switch file {

		case "authentication.json":
			System.File.Authentication = filename
		case "pms.json":
			System.File.PMS = filename
		case "settings.json":
			System.File.Settings = filename
		case "xepg.json":
			System.File.XEPG = filename
		case "urls.json":
			System.File.URLS = filename

		}

	}

	return
}

func updateUrlsJson() {

	getProviderData("m3u", "")
	getProviderData("hdhr", "")

	if Settings.EpgSource == "XEPG" {
		getProviderData("xmltv", "")
	}
	err := buildDatabaseDVR()
	if err != nil {
		ShowError(err, 0)
		return
	}

	buildXEPG(false)
}

// Einstellungen laden und default Werte setzen (Threadfin)
func loadSettings() (settings SettingsStruct, err error) {

	settingsMap, err := loadJSONFileToMap(System.File.Settings)
	if err != nil {
		return SettingsStruct{}, err
	}

	// Deafult Werte setzten
	var defaults = make(map[string]interface{})
	var dataMap = make(map[string]interface{})

	dataMap["xmltv"] = make(map[string]interface{})
	dataMap["m3u"] = make(map[string]interface{})
	dataMap["hdhr"] = make(map[string]interface{})

	defaults["api"] = false
	defaults["authentication.api"] = false
	defaults["authentication.m3u"] = false
	defaults["authentication.pms"] = false
	defaults["authentication.web"] = false
	defaults["authentication.xml"] = false
	defaults["backup.keep"] = 10
	defaults["backup.path"] = System.Folder.Backup
	defaults["buffer"] = "-"
	defaults["buffer.size.kb"] = 1024
	defaults["buffer.timeout"] = 500
	defaults["buffer.autoReconnect"] = false
	defaults["cache.images"] = false
	defaults["epgSource"] = "PMS"
	defaults["ffmpeg.options"] = System.FFmpeg.DefaultOptions
	defaults["vlc.options"] = System.VLC.DefaultOptions
	defaults["files"] = dataMap
	defaults["files.update"] = true
	defaults["filter"] = make(map[string]interface{})
	defaults["git.branch"] = System.Branch
	defaults["language"] = "en"
	defaults["log.entries.ram"] = 500
	defaults["mapping.first.channel"] = 1000
	defaults["xepg.replace.missing.images"] = true
	defaults["xepg.replace.channel.title"] = false
	defaults["m3u8.adaptive.bandwidth.mbps"] = 10
	defaults["port"] = "34400"
	defaults["ssdp"] = true
	defaults["storeBufferInRAM"] = true
	defaults["omitPorts"] = false
	defaults["bindingIPs"] = ""
	defaults["forceHttps"] = false
	defaults["useHttps"] = false
	defaults["threadfinDomain"] = ""
	defaults["enableNonAscii"] = false
	defaults["epgCategories"] = "Kids:kids|News:news|Movie:movie|Series:series|Sports:sports"
	defaults["epgCategoriesColors"] = "kids:mediumpurple|news:tomato|movie:royalblue|series:gold|sports:yellowgreen"
	defaults["tuner"] = 1
	defaults["update"] = []string{"0000"}
	defaults["user.agent"] = System.Name
	defaults["uuid"] = createUUID()
	defaults["udpxy"] = ""
	defaults["version"] = System.DBVersion
	defaults["ThreadfinAutoUpdate"] = true
	if isRunningInContainer() {
		defaults["ThreadfinAutoUpdate"] = false
	}
	defaults["temp.path"] = System.Folder.Temp

	// Default Werte setzen
	for key, value := range defaults {
		if _, ok := settingsMap[key]; !ok {
			settingsMap[key] = value
		}
	}
	err = json.Unmarshal([]byte(mapToJSON(settingsMap)), &settings)
	if err != nil {
		return SettingsStruct{}, err
	}

	// Einstellungen von den Flags übernehmen
	if len(System.Flag.Port) > 0 {
		settings.Port = System.Flag.Port
	}

	if System.Flag.UseHttps {
		settings.UseHttps = System.Flag.UseHttps
	}

	if len(System.Flag.Branch) > 0 {
		settings.Branch = System.Flag.Branch
		ShowInfo(fmt.Sprintf("Git Branch:Switching Git Branch to -> %s", settings.Branch))
	}

	if len(settings.FFmpegPath) == 0 {
		settings.FFmpegPath = searchFileInOS("ffmpeg")
	}

	if len(settings.VLCPath) == 0 {
		settings.VLCPath = searchFileInOS("cvlc")
	}

	// Initialze virutal filesystem for the Buffer
	InitBufferVFS(settings.StoreBufferInRAM)

	settings.Version = System.DBVersion

	err = saveSettings(settings)
	if err != nil {
		return SettingsStruct{}, err
	}

	// Warung wenn FFmpeg nicht gefunden wurde
	if len(Settings.FFmpegPath) == 0 && Settings.Buffer == "ffmpeg" {
		ShowWarning(2020)
	}

	if len(Settings.VLCPath) == 0 && Settings.Buffer == "vlc" {
		ShowWarning(2021)
	}

	// Setzen der globalen Domain
	// Domainnamen setzen
	var domain = ""
	var port = ""
	if settings.UseHttps || settings.ForceClientHttps {
		System.ServerProtocol = "https"
	} else {
		System.ServerProtocol = "http"
	}
	if Settings.ThreadfinDomain != "" {
		domain = Settings.ThreadfinDomain
		if Settings.UseHttps {
			port = Settings.Port
			if port == "" {
				port = "34400"
			}
		} else {
			port = Settings.Port
		}

	} else {
		domain = System.IPAddress
		port = Settings.Port
	}
	if Settings.OmitPorts {
		System.Domain = domain
	} else {
		System.Domain = fmt.Sprintf("%s:%s", domain, port)
	}
	setBaseURL()

	return settings, nil
}

// Einstellungen speichern (Threadfin)
func saveSettings(settings SettingsStruct) (err error) {

	if settings.BackupKeep == 0 {
		settings.BackupKeep = 10
	}

	if len(settings.BackupPath) == 0 {
		settings.BackupPath = System.Folder.Backup
	}

	if settings.BufferTimeout < 0 {
		settings.BufferTimeout = 0
	}

	System.Folder.Temp = settings.TempPath + settings.UUID + string(os.PathSeparator)

	err = writeByteToFile(System.File.Settings, []byte(mapToJSON(settings)))
	if err != nil {
		return
	}

	Settings = settings

	if System.Dev {
		Settings.UUID = "2024-06-DEV-Threadfin!"
	}

	setDeviceID()

	return
}

// Zugriff über die Domain ermöglichen
func setBaseURL() {

	System.BaseURL = fmt.Sprintf("%s://%s", System.ServerProtocol, System.Domain)

	switch Settings.AuthenticationPMS {
	case true:
		System.Addresses.DVR = "username:password@" + System.BaseURL
	case false:
		System.Addresses.DVR = System.BaseURL
	}

	switch Settings.AuthenticationM3U {
	case true:
		System.Addresses.M3U = System.BaseURL + "/m3u/threadfin.m3u?username=xxx&password=yyy"
	case false:
		System.Addresses.M3U = System.BaseURL + "/m3u/threadfin.m3u"
	}

	switch Settings.AuthenticationXML {
	case true:
		System.Addresses.XML = System.BaseURL + "/xmltv/threadfin.xml?username=xxx&password=yyy"
	case false:
		System.Addresses.XML = System.BaseURL + "/xmltv/threadfin.xml"
	}

	if Settings.EpgSource != "XEPG" && !onlyOnce {
		ShowInfo("SOURCE:" + Settings.EpgSource)
		System.Addresses.M3U = getErrMsg(2106)
		System.Addresses.XML = getErrMsg(2106)
		onlyOnce = true
	}
}

// UUID generieren
func createUUID() (uuid string) {
	uuid = time.Now().Format("2006-01") + "-" + randomString(4) + "-" + randomString(6)
	return
}

// Eindeutige Geräte ID für Plex generieren
func setDeviceID() {

	var id = Settings.UUID

	switch Settings.Tuner {
	case 1:
		System.DeviceID = id

	default:
		System.DeviceID = fmt.Sprintf("%s:%d", id, Settings.Tuner)
	}
}

// Provider Streaming-URL zu Threadfin Streaming-URL konvertieren
func createStreamingURL(playlistID, channelNumber, channelName, url string, backup_url_1 string, backup_url_2 string, backup_url_3 string) (streamingURL string, err error) {

	var streamInfo StreamInfo

	if len(Data.Cache.StreamingURLS) == 0 {
		Data.Cache.StreamingURLS = make(map[string]StreamInfo)
	}

	var urlID = getMD5(fmt.Sprintf("%s-%s", playlistID, url))

	if s, ok := Data.Cache.StreamingURLS[urlID]; ok {
		streamInfo = s
	} else {
		streamInfo.URL = url
		streamInfo.BackupChannel1URL = backup_url_1
		streamInfo.BackupChannel2URL = backup_url_2
		streamInfo.BackupChannel3URL = backup_url_3
		streamInfo.Name = channelName
		streamInfo.PlaylistID = playlistID
		streamInfo.ChannelNumber = channelNumber
		streamInfo.URLid = urlID

		Data.Cache.StreamingURLS[urlID] = streamInfo

	}

	streamingURL = System.BaseURL + "/stream/" + streamInfo.URLid
	return
}

func getStreamInfo(urlID string) (streamInfo StreamInfo, err error) {

	if len(Data.Cache.StreamingURLS) == 0 {

		tmp, err := loadJSONFileToMap(System.File.URLS)
		if err != nil {
			return streamInfo, err
		}

		err = json.Unmarshal([]byte(mapToJSON(tmp)), &Data.Cache.StreamingURLS)
		if err != nil {
			return streamInfo, err
		}

	}

	if s, ok := Data.Cache.StreamingURLS[urlID]; ok {
		s.URL = strings.Trim(s.URL, "\r\n")
		s.BackupChannel1URL = strings.Trim(s.BackupChannel1URL, "\r\n")
		s.BackupChannel2URL = strings.Trim(s.BackupChannel2URL, "\r\n")
		s.BackupChannel3URL = strings.Trim(s.BackupChannel3URL, "\r\n")

		streamInfo = s
	} else {
		err = errors.New("streaming error")
	}

	return
}

func isRunningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err != nil {
		return false
	}
	return true
}
