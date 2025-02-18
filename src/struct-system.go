package src

import "threadfin/src/internal/imgcache"

// SystemStruct : Beinhaltet alle Systeminformationen
type SystemStruct struct {
	Addresses struct {
		DVR string
		M3U string
		XML string
	}

	APIVersion             string
	AppName                string
	ARCH                   string
	BackgroundProcess      bool
	Beta                   bool
	Branch                 string
	Build                  string
	Compatibility          string
	ConfigurationWizard    bool
	DBVersion              string
	Dev                    bool
	DeviceID               string
	Domain                 string
	PlexChannelLimit       int
	UnfilteredChannelLimit int

	FFmpeg struct {
		DefaultOptions string
		Path           string
	}

	VLC struct {
		DefaultOptions string
		Path           string
	}

	File struct {
		Authentication string
		M3U            string
		PMS            string
		Settings       string
		URLS           string
		XEPG           string
		XML            string
	}

	Compressed struct {
		GZxml string
	}

	Flag struct {
		Branch   string
		Debug    int
		Info     bool
		Port     string
		UseHttps bool
		Restore  string
		SSDP     bool
	}

	Folder struct {
		Backup       string
		Cache        string
		Config       string
		Custom       string
		Data         string
		ImagesCache  string
		ImagesUpload string
		Temp         string
		Video        string
	}

	BaseURL                string
	ServerProtocol         string
	Hostname               string
	ImageCachingInProgress int
	IPAddress              string
	IPAddressesList        []string
	IPAddressesV4          []string
	IPAddressesV6          []string
	Name                   string
	OS                     string
	ScanInProgress         int
	TimeForAutoUpdate      string

	Notification map[string]Notification

	GitHub struct {
		Branch  string
		Repo    string
		Update  bool
		User    string
		TagName string
	}

	Update Update

	URLBase string
	UDPxy   string
	Version string
	WEB     struct {
		Menu []string
	}
}

type Update struct {
	Git    string
	Name   string
	Github string
}

// GitStruct : Updateinformationen von GitHub
type GitStruct struct {
	Filename string `json:"filename"`
	Version  string `json:"version"`
}



// DataStruct : Alle Daten werden hier abgelegt. (Lineup, XMLTV)
type DataStruct struct {
	Cache struct {
		Images *imgcache.ImageCache
		PMS    map[string]string

		StreamingURLS map[string]*StreamInfo
		XMLTV         map[string]XMLTV

		Streams struct {
			Active []string
		}
	}

	Filter []Filter

	Playlist struct {
		M3U struct {
			Groups struct {
				Text  []string
				Value []string
			}
		}
	}

	StreamPreviewUI struct {
		Active   []string
		Inactive []string
	}

	Streams struct {
		Active   []interface{}
		All      []interface{}
		Inactive []interface{}
	}

	XMLTV struct {
		Files   []string
		Mapping map[string]interface{}
	}

	XEPG struct {
		Channels  map[string]interface{}
		XEPGCount int64
	}
}

// Filter : Wird für die Filterregeln verwendet
type Filter struct {
	CaseSensitive bool
	Rule          string
	Type          string
}

// XEPGChannelStruct : XEPG Struktur
type XEPGChannelStruct struct {
	FileM3UID          string `json:"_file.m3u.id"`
	FileM3UName        string `json:"_file.m3u.name"`
	FileM3UPath        string `json:"_file.m3u.path"`
	GroupTitle         string `json:"group-title"`
	Name               string `json:"name"`
	TvgID              string `json:"tvg-id"`
	TvgLogo            string `json:"tvg-logo"`
	TvgName            string `json:"tvg-name"`
	TvgChno            string `json:"tvg-chno"`
	URL                string `json:"url"`
	UUIDKey            string `json:"_uuid.key"`
	UUIDValue          string `json:"_uuid.value,omitempty"`
	Values             string `json:"_values"`
	XActive            bool   `json:"x-active"`
	XCategory          string `json:"x-category"`
	XChannelID         string `json:"x-channelID"`
	XEPG               string `json:"x-epg"`
	XGroupTitle        string `json:"x-group-title"`
	XMapping           string `json:"x-mapping"`
	XmltvFile          string `json:"x-xmltv-file"`
	XPpvExtra          string `json:"x-ppv-extra"`
	XBackupChannel1    string `json:"x-backup-channel-1"`
	XBackupChannel2    string `json:"x-backup-channel-2"`
	XBackupChannel3    string `json:"x-backup-channel-3"`
	XHideChannel       bool   `json:"x-hide-channel"`
	XName              string `json:"x-name"`
	XUpdateChannelIcon bool   `json:"x-update-channel-icon"`
	XUpdateChannelName bool   `json:"x-update-channel-name"`
	XDescription       string `json:"x-description"`
	Live               bool   `json:"live"`
	IsBackupChannel    bool   `json:"is_backup_channel"`
	BackupChannel1URL  string `json:"backup_channel_1_url"`
	BackupChannel2URL  string `json:"backup_channel_2_url"`
	BackupChannel3URL  string `json:"backup_channel_3_url"`
}

// M3UChannelStructXEPG : M3U Struktur für XEPG
type M3UChannelStructXEPG struct {
	FileM3UID   string `json:"_file.m3u.id"`
	FileM3UName string `json:"_file.m3u.name"`
	FileM3UPath string `json:"_file.m3u.path"`
	GroupTitle  string `json:"group-title"`
	Name        string `json:"name"`
	TvgID       string `json:"tvg-id"`
	TvgLogo     string `json:"tvg-logo"`
	TvgChno     string `json:"tvg-chno"`
	TvgName     string `json:"tvg-name"`
	URL         string `json:"url"`
	UUIDKey     string `json:"_uuid.key"`
	UUIDValue   string `json:"_uuid.value"`
	Values      string `json:"_values"`
}

// FilterStruct : Filter Struktur
type FilterStruct struct {
	Active         bool   `json:"active"`
	CaseSensitive  bool   `json:"caseSensitive"`
	Description    string `json:"description"`
	Exclude        string `json:"exclude"`
	Filter         string `json:"filter"`
	Include        string `json:"include"`
	Name           string `json:"name"`
	Rule           string `json:"rule,omitempty"`
	Type           string `json:"type"`
	StartingNumber string `json:"startingNumber"`
	Category       string `json:"x-category"`
}

// StreamingURLS : Informationen zu allen streaming URL's
type StreamingURLS struct {
	Streams map[string]StreamInfo `json:"channels"`
}

// StreamInfo : Informationen zum Kanal für die streaming URL
type StreamInfo struct {
	ChannelNumber     string `json:"channelNumber"`
	Name              string `json:"name"`
	PlaylistID        string `json:"playlistID"`
	URL               string `json:"url"`
	BackupChannel1URL string `json:"backup_channel_1_url"`
	BackupChannel2URL string `json:"backup_channel_2_url"`
	BackupChannel3URL string `json:"backup_channel_3_url"`
	URLid             string `json:"urlID"`
	HTTP_HEADER		  map[string]string
}

// Notification : Notifikationen im Webinterface
type Notification struct {
	Headline string `json:"headline"`
	Message  string `json:"message"`
	New      bool   `json:"new"`
	Time     string `json:"time"`
	Type     string `json:"type"`
}

// SettingsStruct : Inhalt der settings.json
type SettingsStruct struct {
	API               bool       `json:"api"`
	AuthenticationAPI bool       `json:"authentication.api"`
	AuthenticationM3U bool       `json:"authentication.m3u"`
	AuthenticationPMS bool       `json:"authentication.pms"`
	AuthenticationWEB bool       `json:"authentication.web"`
	AuthenticationXML bool       `json:"authentication.xml"`
	BackupKeep        int        `json:"backup.keep"`
	BackupPath        string     `json:"backup.path"`
	Branch            string     `json:"git.branch,omitempty"`
	Buffer            string     `json:"buffer"`
	BufferSize        int        `json:"buffer.size.kb"`
	BufferTimeout     float64    `json:"buffer.timeout"`
	BufferAutoReconnect bool     `json:"buffer.autoReconnect"`
	BufferTerminationTimeout int `json:"buffer.terminationTimeout"`
	CacheImages       bool       `json:"cache.images"`
	ChangeVersion     bool       `json:"changeVersion"`
	EpgSource         string     `json:"epgSource"`
	FFmpegOptions     string     `json:"ffmpeg.options"`
	FFmpegPath        string     `json:"ffmpeg.path"`
	VLCOptions        string     `json:"vlc.options"`
	VLCPath           string     `json:"vlc.path"`
	FileM3U           []string   `json:"file,omitempty"`  // Beim Wizard wird die M3U in ein Slice gespeichert
	FileXMLTV         []string   `json:"xmltv,omitempty"` // Altes Speichersystem der Provider XML Datei Slice (Wird für die Umwandlung auf das neue benötigt)

	Files struct {
		HDHR  map[string]interface{} `json:"hdhr"`
		M3U   map[string]interface{} `json:"m3u"`
		XMLTV map[string]interface{} `json:"xmltv"`
	} `json:"files"`

	FilesUpdate               bool                  `json:"files.update"`
	Filter                    map[int64]interface{} `json:"filter"`
	Key                       string                `json:"key,omitempty"`
	WebClientLanguage         string                `json:"webclient.language"`
	LogEntriesRAM             int                   `json:"log.entries.ram"`
	M3U8AdaptiveBandwidthMBPS int                   `json:"m3u8.adaptive.bandwidth.mbps"`
	MappingFirstChannel       float64               `json:"mapping.first.channel"`
	Port                      string                `json:"port"`
	SSDP                      bool                  `json:"ssdp"`
	TempPath                  string                `json:"temp.path"`
	Tuner                     int                   `json:"tuner"`
	Update                    []string              `json:"update"`
	UpdateURL                 string                `json:"update.url,omitempty"`
	UserAgent                 string                `json:"user.agent"`
	UUID                      string                `json:"uuid"`
	UDPxy                     string                `json:"udpxy"`
	Version                   string                `json:"version"`
	XepgReplaceMissingImages  bool                  `json:"xepg.replace.missing.images"`
	XepgReplaceChannelTitle   bool                  `json:"xepg.replace.channel.title"`
	ThreadfinAutoUpdate       bool                  `json:"ThreadfinAutoUpdate"`
	StoreBufferInRAM          bool                  `json:"storeBufferInRAM"`
	OmitPorts                 bool                  `json:"omitPorts"`
	BindingIPs                string                `json:"bindingIPs"`
	ForceHttpsToUpstream      bool                  `json:"forceHttps"`
	UseHttps                  bool                  `json:"useHttps"`
	ForceClientHttps          bool                  `json:"forceClientHttps"`
	ThreadfinDomain           string                `json:"threadfinDomain"`
	EnableNonAscii            bool                  `json:"enableNonAscii"`
	EpgCategories             string                `json:"epgCategories"`
	EpgCategoriesColors       string                `json:"epgCategoriesColors"`
	Dummy                     bool                  `json:"dummy"`
	DummyChannel              string                `json:"dummyChannel"`
	IgnoreFilters             bool                  `json:"ignoreFilters"`
}

// LanguageUI : Sprache für das WebUI
type LanguageUI struct {
	Login struct {
		Failed string
	}
}
