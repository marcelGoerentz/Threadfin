package src

// RequestStruct : Anfragen über die Websocket Schnittstelle
type RequestStruct struct {
	// Befehle an Threadfin
	Cmd string `json:"cmd"`

	// Benutzer
	DeleteUser bool                   `json:"deleteUser,omitempty"`
	UserData   map[string]interface{} `json:"userData,omitempty"`

	// Mapping
	EpgMapping map[string]interface{} `json:"epgMapping,omitempty"`

	// Restore
	Base64 string `json:"base64,omitempty"`

	// Neue Werte für die Einstellungen (settings.json)
	Settings struct {
		API                      *bool     `json:"api,omitempty"`
		SSDP                     *bool     `json:"ssdp,omitempty"`
		AuthenticationAPI        *bool     `json:"authentication.api,omitempty"`
		AuthenticationM3U        *bool     `json:"authentication.m3u,omitempty"`
		AuthenticationPMS        *bool     `json:"authentication.pms,omitempty"`
		AuthenticationWEP        *bool     `json:"authentication.web,omitempty"`
		AuthenticationXML        *bool     `json:"authentication.xml,omitempty"`
		BackupKeep               *int      `json:"backup.keep,omitempty"`
		BackupPath               *string   `json:"backup.path,omitempty"`
		Buffer                   *string   `json:"buffer,omitempty"`
		BufferSize               *int      `json:"buffer.size.kb,omitempty"`
		BufferTimeout            *float64  `json:"buffer.timeout,omitempty"`
		CacheImages              *bool     `json:"cache.images,omitempty"`
		EpgSource                *string   `json:"epgSource,omitempty"`
		FFmpegOptions            *string   `json:"ffmpeg.options,omitempty"`
		FFmpegPath               *string   `json:"ffmpeg.path,omitempty"`
		VLCOptions               *string   `json:"vlc.options,omitempty"`
		VLCPath                  *string   `json:"vlc.path,omitempty"`
		FilesUpdate              *bool     `json:"files.update,omitempty"`
		TempPath                 *string   `json:"temp.path,omitempty"`
		Tuner                    *int      `json:"tuner,omitempty"`
		UDPxy                    *string   `json:"udpxy,omitempty"`
		Update                   *[]string `json:"update,omitempty"`
		UserAgent                *string   `json:"user.agent,omitempty"`
		XepgReplaceMissingImages *bool     `json:"xepg.replace.missing.images,omitempty"`
		XepgReplaceChannelTitle  *bool     `json:"xepg.replace.channel.title,omitempty"`
		ThreadfinAutoUpdate      *bool     `json:"ThreadfinAutoUpdate,omitempty"`
		SchemeM3U                *string   `json:"scheme.m3u,omitempty"`
		SchemeXML                *string   `json:"scheme.xml,omitempty"`
		StoreBufferInRAM         *bool     `json:"storeBufferInRAM,omitempty"`
		OmitPorts                *bool     `json:"omitPorts,omitempty"`
		BindingIPs               *string   `json:"bindingIPs,omitempty"`
		ForceHttpsToUpstream     *bool     `json:"forceHttps,omitempty"`
		UseHttps                 *bool     `json:"useHttps,omitempty"`
		ForceClientHttps         *bool     `json:"forceClientHttps"`
		ThreadfinDomain          *string   `json:"threadfinDomain,omitempty"`
		EnableNonAscii           *bool     `json:"enableNonAscii,omitempty"`
		EpgCategories            *string   `json:"epgCategories,omitempty"`
		EpgCategoriesColors      *string   `json:"epgCategoriesColors,omitempty"`
		Dummy                    *bool     `json:"dummy,omitempty"`
		DummyChannel             *string   `json:"dummyChannel,omitempty"`
		IgnoreFilters            *bool     `json:"ignoreFilters,omitempty"`
	} `json:"settings,omitempty"`

	// Upload Logo
	Filename string `json:"filename,omitempty"`

	// Filter
	Filter map[int64]interface{} `json:"filter,omitempty"`

	// Dateien (M3U, HDHR, XMLTV)
	Files struct {
		HDHR  map[string]interface{} `json:"hdhr,omitempty"`
		M3U   map[string]interface{} `json:"m3u,omitempty"`
		XMLTV map[string]interface{} `json:"xmltv,omitempty"`
	} `json:"files,omitempty"`

	// Wizard
	Wizard struct {
		EpgSource *string `json:"epgSource,omitempty"`
		M3U       *string `json:"m3u,omitempty"`
		Tuner     *int    `json:"tuner,omitempty"`
		XMLTV     *string `json:"xmltv,omitempty"`
	} `json:"wizard,omitempty"`
}

// ResponseStruct : Antworten an den Client (WEB)
type ResponseStruct struct {
	ClientInfo struct {
		ARCH           string   `json:"arch"`
		Branch         string   `json:"branch,omitempty"`
		DVR            string   `json:"DVR"`
		EpgSource      string   `json:"epgSource"`
		Errors         int      `json:"errors"`
		M3U            string   `json:"m3u-url"`
		OS             string   `json:"os"`
		Streams        string   `json:"streams"`
		ActiveClients  int      `json:"activeClients"`
		TotalClients   int      `json:"totalClients"`
		ActivePlaylist int      `json:"activePlaylist"`
		TotalPlaylist  int      `json:"totalPlaylist"`
		SystemIPs	   []string `json:"systemIPs"`
		UUID           string   `json:"uuid"`
		Version        string   `json:"version"`
		Warnings       int      `json:"warnings"`
		XEPGCount      int64    `json:"xepg"`
		XML            string   `json:"xepg-url"`
	} `json:"clientInfo,omitempty"`

	Data struct {
		Playlist struct {
			M3U struct {
				Groups struct {
					Text  []string `json:"text"`
					Value []string `json:"value"`
				} `json:"groups"`
			} `json:"m3u"`
		} `json:"playlist"`

		StreamPreviewUI struct {
			Active   []string `json:"activeStreams"`
			Inactive []string `json:"inactiveStreams"`
		}
	} `json:"data"`

	Alert               string                 `json:"alert,omitempty"`
	ConfigurationWizard bool                   `json:"configurationWizard"`
	Error               string                 `json:"err,omitempty"`
	Log                 WebScreenLogStruct     `json:"log"`
	LogoURL             string                 `json:"logoURL,omitempty"`
	OpenLink            string                 `json:"openLink,omitempty"`
	OpenMenu            string                 `json:"openMenu,omitempty"`
	Reload              bool                   `json:"reload,omitempty"`
	Settings            SettingsStruct         `json:"settings"`
	Status              bool                   `json:"status"`
	Token               string                 `json:"token,omitempty"`
	Users               map[string]interface{} `json:"users,omitempty"`
	Wizard              int                    `json:"wizard,omitempty"`
	XEPG                map[string]interface{} `json:"xepg"`

	Notification map[string]Notification `json:"notification,omitempty"`
}

// APIRequestStruct : Anfrage über die API Schnittstelle
type APIRequestStruct struct {
	Cmd      string `json:"cmd"`
	Password string `json:"password"`
	Token    string `json:"token"`
	Username string `json:"username"`
}

// APIResponseStruct : Antwort an den Client (API)
type APIResponseStruct struct {
	Error      string                 `json:"error,omitempty"`
	SystemInfo    *SystemInfoStruct    `json:"systemInfo,omitempty"`
	ActiveStreams *ActiveStreamsStruct `json:"activeStreams,omitempty"`
	Token      string                 `json:"token,omitempty"`
}

type ActiveStreamsStruct struct {
	Playlists []PlaylistStruct `json:"playlists"`
}

type PlaylistStruct struct {
	PlaylistName string   `json:"playlistName"`
	ChannelList  []string `json:"channelList"`
}

type SystemInfoStruct struct {
	ThreadfinVersion string            `json:"appVersion"`
	APIVersion       string            `json:"apiVersion"`
	EpgSource		 string            `json:"epgSource"`
	SystemURLs       SystemURLsStruct  `json:"systemURLs"`
	ChannelInfo      ChannelInfoStruct `json:"channelInfo"`
}

type SystemURLsStruct struct {
	DVR  string `json:"dvr"`
	M3U  string `json:"m3u"`
    XEPG string `json:"xepg"`
}

type ChannelInfoStruct struct {
	ActiveChannels uint32 `json:"activeChannels"`
	AllChannels    uint32 `json:"allChannels"`
	XEPGChannels   uint32 `json:"xepgChannels"`
}

// WebScreenLogStruct : Logs werden im RAM gespeichert und für das Webinterface bereitgestellt
type WebScreenLogStruct struct {
	Errors   int      `json:"errors"`
	Log      []string `json:"log"`
	Warnings int      `json:"warnings"`
}
