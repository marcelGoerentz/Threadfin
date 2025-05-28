package src

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	//"github.com/avfs/avfs"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// System : Beinhaltet alle Systeminformationen
var System SystemStruct

// WebScreenLog : Logs werden im RAM gespeichert und für das Webinterface bereitgestellt
var WebScreenLog WebScreenLogStruct

// Settings : Inhalt der settings.json
var Settings SettingsStruct

// Data : Alle Daten werden hier abgelegt. (Lineup, XMLTV)
var Data DataStruct

// SystemFiles : Alle Systemdateien
var SystemFiles = []string{"authentication.json", "pms.json", "settings.json", "xepg.json", "urls.json"}

// bufferVFS : Filesystem to use for the Buffer
//var bufferVFS avfs.VFS

// Lock : Lock Map
var Lock = sync.RWMutex{}

// Init : Systeminitialisierung
func Init() (err error) {

	var debug string

	// System Einstellungen
	System.AppName = strings.ToLower(System.Name)
	System.ARCH = runtime.GOARCH
	System.OS = runtime.GOOS
	System.PlexChannelLimit = 480
	System.UnfilteredChannelLimit = 480
	System.Compatibility = "0.1.0"

	// FFmpeg Default Einstellungen
	System.FFmpeg.DefaultOptions = "-hide_banner -loglevel error -i [URL] -c copy -f mpegts pipe:1"
	System.VLC.DefaultOptions = "-I dummy [URL] --sout \"#std{mux=ts,access=file,dst=-}\" --no-sout-all"

	// Default Logeinträge, wird später von denen aus der settings.json überschrieben. Muss gemacht werden, damit die ersten Einträge auch im Log (webUI aangezeigt werden)
	Settings.LogEntriesRAM = 500

	// Variablen für den Update Prozess
	//System.Update.Git = "https://github.com/Threadfin/Threadfin/blob"
	System.Update.Git = fmt.Sprintf("https://github.com/%s/%s", System.GitHub.User, System.GitHub.Repo)
	System.Update.Github = fmt.Sprintf("https://api.github.com/repos/%s/%s", System.GitHub.User, System.GitHub.Repo)
	System.Update.Name = "Threadfin"

	// Ordnerpfade festlegen
	var tempFolder = os.TempDir() + string(os.PathSeparator) + System.AppName + string(os.PathSeparator)

	if len(System.Folder.Config) == 0 {
		System.Folder.Config = GetUserHomeDirectory() + string(os.PathSeparator) + "." + System.AppName + string(os.PathSeparator)
	} else {
		System.Folder.Config = strings.TrimRight(System.Folder.Config, string(os.PathSeparator)) + string(os.PathSeparator)
	}

	System.Folder.Config = getPlatformPath(System.Folder.Config) + string(os.PathSeparator)

	System.Folder.Backup = System.Folder.Config + "backup" + string(os.PathSeparator)
	System.Folder.Data = System.Folder.Config + "data" + string(os.PathSeparator)
	System.Folder.Cache = System.Folder.Config + "cache" + string(os.PathSeparator)
	System.Folder.ImagesCache = System.Folder.Cache + "images" + string(os.PathSeparator)
	System.Folder.ImagesUpload = System.Folder.Data + "images" + string(os.PathSeparator)
	System.Folder.Custom = System.Folder.ImagesUpload + "custom" + string(os.PathSeparator)
	System.Folder.Video = System.Folder.Data + "video" + string(os.PathSeparator)
	System.Folder.Temp = tempFolder

	// Dev Info
	showDevInfo()

	// System Ordner erstellen
	err = createSystemFolders()
	if err != nil {
		ShowError(err, 1070)
		return
	}

	if len(System.Flag.Restore) > 0 {
		// Einstellungen werden über CLI wiederhergestellt. Weitere Initialisierung ist nicht notwendig.
		return
	}

	System.File.XML = getPlatformFile(fmt.Sprintf("%s%s.xml", System.Folder.Data, System.AppName))
	System.File.M3U = getPlatformFile(fmt.Sprintf("%s%s.m3u", System.Folder.Data, System.AppName))

	System.Compressed.GZxml = getPlatformFile(fmt.Sprintf("%s%s.xml.gz", System.Folder.Data, System.AppName))

	err = activatedSystemAuthentication()
	if err != nil {
		return
	}

	err = resolveHostIP()
	if err != nil {
		ShowError(err, 1002)
	}

	// Menü für das Webinterface
	System.WEB.Menu = []string{"playlist", "xmltv", "filter", "mapping", "users", "settings", "log", "logout"}

	ShowInfo(fmt.Sprintf("Info:For help run: %s %s", getPlatformFile(os.Args[0]), " -h"))

	// Überprüfen ob Threadfin als root läuft
	if os.Geteuid() == 0 {
		ShowWarning(2110)
	}

	if System.Flag.Debug > 0 {
		debug = fmt.Sprintf("Debug Level:%d", System.Flag.Debug)
		ShowDebug(debug, 1)
	}

	ShowInfo(fmt.Sprintf("Version:%s Build: %s", System.Version, System.Build))
	ShowInfo(fmt.Sprintf("Database Version:%s", System.DBVersion))
	ShowInfo(fmt.Sprintf("System IP Addresses:IPv4: %d | IPv6: %d", len(System.IPAddressesV4), len(System.IPAddressesV6)))
	ShowInfo("Hostname:" + System.Hostname)
	ShowInfo(fmt.Sprintf("System Folder:%s", getPlatformPath(System.Folder.Config)))

	// Systemdateien erstellen (Falls nicht vorhanden)
	err = createSystemFiles()
	if err != nil {
		ShowError(err, 1071)
		return
	}
	
	// Einstellungen laden (settings.json)
	ShowInfo(fmt.Sprintf("Load Settings:%s", System.File.Settings))

	_, err = loadSettings()
	if err != nil {
		ShowError(err, 0)
		return
	}

	// Berechtigung aller Ordner überprüfen
	err = checkFilePermission(System.Folder.Config)
	if err != nil {
		ShowError(err, 1015)
	}

	// Separaten tmp Ordner für jede Instanz
	//System.Folder.Temp = System.Folder.Temp + Settings.UUID + string(os.PathSeparator)
	ShowInfo(fmt.Sprintf("Temporary Folder:%s", getPlatformPath(System.Folder.Temp)))

	err = checkFolder(System.Folder.Temp)
	if err != nil {
		return
	}

	err = checkFilePermission(System.Folder.Temp)
	if err != nil {
		ShowError(err, 1016)
	}

	err = removeChildItems(getPlatformPath(System.Folder.Temp))
	if err != nil {
		return
	}

	// Branch festlegen
	System.Branch = cases.Title(language.English).String(Settings.Branch)

	if System.Dev {
		System.Branch = cases.Title(language.English).String("development")
	}

	if len(System.Branch) == 0 {
		System.Branch = cases.Title(language.English).String("main")
	}

	ShowInfo(fmt.Sprintf("GitHub:https://github.com/%s", System.GitHub.User))
	ShowInfo(fmt.Sprintf("Git Branch:%s [%s]", System.Branch, System.GitHub.User))

	System.URLBase = fmt.Sprintf("%s://%s:%s", System.ServerProtocol, System.IPAddress, Settings.Port)

	/*
	// HTML Dateien erstellen, mit dev == true werden die lokalen HTML Dateien verwendet
	if System.Dev {

		HTMLInit("webUI", "src", strings.Join([]string{"web", "public"}, string(os.PathSeparator)), "src"+string(os.PathSeparator)+"webUI.go")
		err = BuildGoFile()
		if err != nil {
			return
		}

	}*/

	// DLNA Server starten
	if Settings.SSDP {
		err = SSDP()
		if err != nil {
			return
		}
	}

	// HTML Datein laden
	loadHTMLMap()

	return
}

// StartSystem : System wird gestartet
func StartSystem(updateProviderFiles bool) (err error) {

	setDeviceID()

	if System.ScanInProgress == 1 {
		return
	}

	// Systeminformationen in der Konsole ausgeben
	ShowInfo(fmt.Sprintf("UUID:%s", Settings.UUID))
	ShowInfo(fmt.Sprintf("Tuner (Plex / Emby):%d", Settings.Tuner))
	ShowInfo(fmt.Sprintf("EPG Source:%s", Settings.EpgSource))
	ShowInfo(fmt.Sprintf("Plex Channel Limit:%d", System.PlexChannelLimit))
	ShowInfo(fmt.Sprintf("Unfiltered Chan. Limit:%d", System.UnfilteredChannelLimit))

	// Providerdaten aktualisieren
	if len(Settings.Files.M3U) > 0 && Settings.FilesUpdate || updateProviderFiles {

		err = ThreadfinAutoBackup()
		if err != nil {
			ShowError(err, 1090)
		}

		getProviderData("m3u", "")
		getProviderData("hdhr", "")

		if Settings.EpgSource == "XEPG" {
			getProviderData("xmltv", "")
		}

	}

	err = buildDatabaseDVR()
	if err != nil {
		ShowError(err, 0)
		return
	}

	buildXEPG(true)

	return
}
