package src

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"threadfin/src/internal/authentication"

	"github.com/gorilla/websocket"
	"golang.org/x/net/http2"
)

var streamManager = NewStreamManager()
var webServer *WebServer = nil

// StartWebserver : Startet den Webserver
func (ws *WebServer) StartWebserver() (err error) {

	webServer = ws
	addMimeExtensions()
	ws.SM = streamManager

	var port = Settings.Port
	var serverMux = http.NewServeMux()

	serverMux.HandleFunc("/", Index)
	serverMux.HandleFunc("/stream/", stream)
	serverMux.HandleFunc("/xmltv/", Threadfin)
	serverMux.HandleFunc("/m3u/", Threadfin)
	serverMux.HandleFunc("/ws/", WS)
	serverMux.HandleFunc("/web/", Web)
	serverMux.HandleFunc("/download/", Download)
	serverMux.HandleFunc("/api/", API)
	serverMux.HandleFunc("/images/", Images)
	serverMux.HandleFunc("/data_images/", DataImages)
	serverMux.HandleFunc("/ppv/enable", enablePPV)
	serverMux.HandleFunc("/ppv/disable", disablePPV)
	//serverMux.HandleFunc("/auto/", Auto)

	// Create server structure
	srv := &http.Server{
		Addr:    port,
		Handler: serverMux,
	}

	ws.Server = srv

	regexIpV4, _ := regexp.Compile(`(?:\d{1,3}\.){3}\d{1,3}`)
	regexIpV6, _ := regexp.Compile(`(?:[A-Fa-f0-9]{0,4}:){3,7}[a-fA-F0-9]{1,4}`)
	var customIps []string
	var customIpsV4 = regexIpV4.FindAllString(Settings.BindingIPs, -1)
	var customIpsV6 = regexIpV6.FindAllString(Settings.BindingIPs, -1)
	if customIpsV4 != nil || customIpsV6 != nil {
		customIps = make([]string, len(customIpsV4)+len(customIpsV6))
		copy(customIps, customIpsV4)
		copy(customIps[len(customIpsV4):], customIpsV6)
	}

	if customIps != nil {
		for _, address := range customIps {
			ShowInfo("DVR IP:" + address + ":" + Settings.Port)
			ShowHighlight(fmt.Sprintf("Web Interface:%s://%s:%s/web/", System.ServerProtocol, address, Settings.Port))
			if Settings.UseHttps {
				go func(address string) {
					//activate HTTP2
					http2.ConfigureServer(srv, &http2.Server{})
					if err = srv.ListenAndServeTLS(System.Folder.Config+"server.crt", System.Folder.Config+"server.key"); err != nil {
						//ShowError(err, 1001)
						return
					}
				}(address)
			} else {
				go func(address string) {
					srv.Addr = address + ":" + port
					if err = srv.ListenAndServe(); err != nil {
						//ShowError(err, 1001)
						return
					}
				}(address)
			}

		}
	} else {
		for _, ip := range System.IPAddressesV4 {
			ShowInfo("DVR IP:" + ip + ":" + Settings.Port)
			ShowHighlight(fmt.Sprintf("Web Interface:%s://%s:%s/web/", System.ServerProtocol, ip, Settings.Port))
		}

		for _, ip := range System.IPAddressesV6 {
			ShowInfo("DVR IP:" + ip + ":" + Settings.Port)
			ShowHighlight(fmt.Sprintf("Web Interface:%s://%s:%s/web/", System.ServerProtocol, ip, Settings.Port))
		}
		if Settings.UseHttps {
			go func() {
				//activate HTTP2
				http2.ConfigureServer(srv, &http2.Server{})
				if err = srv.ListenAndServeTLS(System.Folder.Config+"server.crt", System.Folder.Config+"server.key"); err != nil {
					//ShowError(err, 1001)
					return
				}
			}()
		} else {
			go func() {
				srv.Addr = ":" + port
				if err = srv.ListenAndServe(); err != nil {
					//ShowError(err, 1001)
					return
				}
			}()
		}
	}
	return
}

func (ws *WebServer) restartWebserver() error {
	for _, playlist := range streamManager.Playlists {
		for streamID, stream := range playlist.Streams {
			stream.StopStream(streamID)
		}
	}
	ShutDownWebserver(ws)
	return ws.StartWebserver()

}

func ShutDownWebserver(ws *WebServer) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	err := ws.Server.Shutdown(ctx)
	if err != nil {
		ShowError(err, 7000)
	}
}

func addMimeExtensions() {
	mime.AddExtensionType(".m3u8", "application/vnc.apple.mpegurl")
	mime.AddExtensionType(".m3u8", "application/x-mpegURL")
	mime.AddExtensionType(".m3u", "application/m3u")
	mime.AddExtensionType(".m3u", "audio/m3u")
	mime.AddExtensionType(".m3u", "audio/x-m3u")
	mime.AddExtensionType(".m3u", "audio/x-mp3-playlist")
}

// Index : Web Server /
func Index(w http.ResponseWriter, r *http.Request) {

	var err error
	var response []byte
	var path = r.URL.Path
	var debug = fmt.Sprintf("Web Server Request:Path: %s", path)

	ShowDebug(debug, 2)

	switch path {

	case "/discover.json":
		response, err = getDiscover()
		w.Header().Set("Content-Type", "application/json")

	case "/lineup_status.json":
		response, err = getLineupStatus()
		w.Header().Set("Content-Type", "application/json")

	case "/lineup.json":
		if Settings.AuthenticationPMS {

			_, err := basicAuth(r, "authentication.pms")
			if err != nil {
				ShowError(err, 000)
				httpStatusError(w, http.StatusForbidden)
				return
			}

		}

		response, err = getLineup()
		w.Header().Set("Content-Type", "application/json")

	case "/device.xml", "/capability":
		response, err = getCapability()
		w.Header().Set("Content-Type", "application/xml")

	default:
		response, err = getCapability()
		w.Header().Set("Content-Type", "application/xml")
	}

	if err == nil {

		w.WriteHeader(200)
		w.Write(response)
		return

	}

	httpStatusError(w, http.StatusInternalServerError)
}

// stream : Web Server /stream/
func stream(w http.ResponseWriter, r *http.Request) {

	var err error

	var path = strings.Replace(r.RequestURI, "/stream/", "", 1)
	//var stream = strings.SplitN(path, "-", 2)

	streamInfo, err := getStreamInfo(path)
	if err != nil {
		ShowError(err, 1203)
		httpStatusError(w, http.StatusNotFound)
		return
	}

	if r.Method == "HEAD" {
		client := &http.Client{}
		log.Println("URL: ", streamInfo.URL)
		req, err := http.NewRequest("HEAD", streamInfo.URL, nil)
		if err != nil {
			ShowError(err, 1501)
			httpStatusError(w, http.StatusMethodNotAllowed)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			ShowError(err, 1502)
			httpStatusError(w, http.StatusMethodNotAllowed)
			return
		}
		defer resp.Body.Close()

		// Copy headers from the source HEAD response to the outgoing response
		log.Println("HEAD response from source: ", resp.Header)
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		return
	}

	if Settings.ForceHttpsToUpstream {
		var u *url.URL
		u, err = url.Parse(streamInfo.URL)
		setProtocol(streamInfo, u, err)
		if streamInfo.BackupChannel1URL != "" {
			u, err = url.Parse(streamInfo.BackupChannel1URL)
			setProtocol(streamInfo, u, err)
		}
		if streamInfo.BackupChannel1URL != "" {
			u, err = url.Parse(streamInfo.BackupChannel2URL)
			setProtocol(streamInfo, u, err)
		}
		if streamInfo.BackupChannel1URL != "" {
			u, err = url.Parse(streamInfo.BackupChannel3URL)
			setProtocol(streamInfo, u, err)
		}
	}

	// If an UDPxy host is set, and the stream URL is multicast (i.e. starts with 'udp://@'),
	// then streamInfo.URL needs to be rewritten to point to UDPxy.
	if Settings.UDPxy != "" && strings.HasPrefix(streamInfo.URL, "udp://@") {
		streamInfo.URL = fmt.Sprintf("http://%s/udp/%s/", Settings.UDPxy, strings.TrimPrefix(streamInfo.URL, "udp://@"))
	}

	switch Settings.Buffer {

	case "-":
		ShowInfo(fmt.Sprintf("Buffer:false [%s]", Settings.Buffer))

	case "threadfin":
		if strings.Contains(streamInfo.URL, "rtsp://") || strings.Contains(streamInfo.URL, "rtp://") {
			err = errors.New("RTSP and RTP streams are not supported")
			ShowError(err, 2004)

			ShowInfo("Streaming URL:" + streamInfo.URL)
			http.Redirect(w, r, streamInfo.URL, http.StatusFound)

			ShowInfo("Streaming Info:URL was passed to the client")
			return
		}

		ShowInfo(fmt.Sprintf("Buffer:true [%s]", Settings.Buffer))

	default:
		ShowInfo(fmt.Sprintf("Buffer:true [%s]", Settings.Buffer))

	}

	if Settings.Buffer != "-" {
		ShowInfo(fmt.Sprintf("Buffer Size:%d KB", Settings.BufferSize))
	}

	log.Println("Stream Info: ", streamInfo)
	log.Println("M3U Info: ", Settings.Files.M3U)

	ShowInfo(fmt.Sprintf("Channel Name:%s", streamInfo.Name))
	ShowInfo(fmt.Sprintf("Client User-Agent:%s", r.Header.Get("User-Agent")))

	// Prüfen ob der Buffer verwendet werden soll
	switch Settings.Buffer {

	case "-":
		providerSettings, ok := Settings.Files.M3U[streamInfo.PlaylistID].(map[string]interface{})
		if !ok {
			return
		}

		proxyIP, ok := providerSettings["http_proxy.ip"].(string)
		if !ok {
			return
		}

		proxyPort, ok := providerSettings["http_proxy.port"].(string)
		if !ok {
			return
		}

		if proxyIP != "" && proxyPort != "" {
			ShowInfo("Streaming Info: Streaming through proxy.")

			proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s", proxyIP, proxyPort))
			if err != nil {
				return
			}

			httpClient := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				},
			}
			resp, err := httpClient.Get(streamInfo.URL)
			if err != nil {
				http.Error(w, "Failed to fetch stream", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}

			w.WriteHeader(resp.StatusCode)
			_, err = io.Copy(w, resp.Body)
			if err != nil {
				http.Error(w, "Failed to stream response", http.StatusInternalServerError)
				return
			}
		} else {
			ShowInfo("Streaming URL:" + streamInfo.URL)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			http.Redirect(w, r, streamInfo.URL, http.StatusFound)

			ShowInfo("Streaming Info:URL was passed to the client.")
			ShowInfo("Streaming Info:Threadfin is no longer involved, the client connects directly to the streaming server.")
		}

	default:
		w.Header().Set("Content-Type", "video/mp2t")
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Keep-Alive", "timeout=10, max=100")
		streamManager.ServeStream(streamInfo, w, r)
	}
}

func setProtocol(streamInfo *StreamInfo, u *url.URL, err error) {
	if err == nil {
		switch u.Scheme {
		case "http":
			u.Scheme = "https"
			streamInfo.URL = u.String()
		case "https", "rtsp", "rtp":
			return
		default:
			ShowInfo(fmt.Sprintf("Streaming: Unknown protocol: %s", u.Scheme))
		}
	}
}

// Auto : HDHR routing (wird derzeit nicht benutzt)
func Auto(w http.ResponseWriter, r *http.Request) {

	var channelID = strings.Replace(r.RequestURI, "/auto/v", "", 1)
	fmt.Println(channelID)

	/*
		switch Settings.Buffer {

		case true:
			var playlistID, streamURL, err = getStreamByChannelID(channelID)
			if err == nil {
				bufferingStream(playlistID, streamURL, w, r)
			} else {
				httpStatusError(w, r, 404)
			}

		case false:
			httpStatusError(w, r, 423)
		}
	*/
}

// Threadfin : Web Server /xmltv/ und /m3u/
func Threadfin(w http.ResponseWriter, r *http.Request) {

	var requestType, groupTitle, file, content, contentType string
	var err error
	var path = strings.TrimPrefix(r.URL.Path, "/")
	var groups = []string{}

	// XMLTV Datei
	if strings.Contains(path, "xmltv/") {

		requestType = "xml"

		file = System.Folder.Data + getFilenameFromPath(path)

		content, err = readStringFromFile(file)
		if err != nil {
			httpStatusError(w, http.StatusNotFound)
			return
		}

	}

	// M3U Datei
	if strings.Contains(path, "m3u/") {

		requestType = "m3u"
		groupTitle = r.URL.Query().Get("group-title")

		m3uFilePath := System.Folder.Data + "threadfin.m3u"

		// Check if the m3u file exists
		if _, err := os.Stat(m3uFilePath); err == nil {
			log.Println("Serving existing m3u file")
			http.ServeFile(w, r, m3uFilePath)
			return
		}

		log.Println("M3U file does not exist, building new one")

		if !System.Dev {
			// false: Dateiname wird im Header gesetzt
			// true: M3U wird direkt im Browser angezeigt
			w.Header().Set("Content-Disposition", "attachment; filename="+getFilenameFromPath(path))
		}

		if len(groupTitle) > 0 {
			groups = strings.Split(groupTitle, ",")
		}

		content, err = buildM3U(groups)
		if err != nil {
			ShowError(err, 000)
		}

	}

	// Authentifizierung überprüfen
	err = urlAuth(r, requestType)
	if err != nil {
		ShowError(err, 000)
		httpStatusError(w, http.StatusForbidden)
		return
	}

	contentType = http.DetectContentType([]byte(content))
	if strings.Contains(strings.ToLower(contentType), "xml") {
		contentType = "application/xml; charset=utf-8"
	}

	w.Header().Set("Content-Type", contentType)
	w.Write([]byte(content))
}

// Images : Image Cache /images/
func Images(w http.ResponseWriter, r *http.Request) {

	var err error
	var path = strings.TrimPrefix(r.URL.Path, "/images/")
	var filePath = System.Folder.ImagesCache + getFilenameFromPath(path)

	content, err := readByteFromFile(filePath)
	if err != nil {
		httpStatusError(w, http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Type", getContentType(filePath))
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(content)))
	w.WriteHeader(200)
	w.Write(content)
}

// DataImages : Image Pfad für Logos / Bilder die hochgeladen wurden /data_images/
func DataImages(w http.ResponseWriter, r *http.Request) {

	var err error
	var path = strings.TrimPrefix(r.URL.Path, "/")
	var filePath = System.Folder.ImagesUpload + getFilenameFromPath(path)

	content, err := readByteFromFile(filePath)
	if err != nil {
		httpStatusError(w, http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Type", getContentType(filePath))
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(content)))
	w.WriteHeader(200)
	w.Write(content)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WS : Web Sockets /ws/
func WS(w http.ResponseWriter, r *http.Request) {

	var err error
	var request RequestStruct
	var response ResponseStruct
	response.Status = true

	var newToken string

	// Upgrade connection to websocket connection
	conn, err := upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		ShowError(err, 0)
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}

	for {

		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			ShowError(err, 11103)
			break
		}

		err = json.Unmarshal(msgBytes, &request)

		if err != nil {
			ShowError(err, 1120)
			break
		}

		if !System.ConfigurationWizard {

			switch Settings.AuthenticationWEB {

			// Token Authentication
			case true:

				var token string
				tokens, ok := r.URL.Query()["Token"]

				if !ok || len(tokens[0]) < 1 {
					token = "-"
				} else {
					token = tokens[0]
				}

				newToken, err = tokenAuthentication(token)
				if err != nil {

					response.Status = false
					response.Reload = true
					response.Error = err.Error()
					request.Cmd = "-"

					if err = conn.WriteJSON(response); err != nil {
						ShowError(err, 1102)
					}

					return
				}

				response.Token = newToken
				response.Users, _ = authentication.GetAllUserData()

			}

		}

		switch request.Cmd {
		// Daten lesen
		case "getServerConfig":
			//response.Config = Settings

		case "updateLog":
			response = setDefaultResponseData(response, false)
			if err = conn.WriteJSON(response); err != nil {
				ShowError(err, 1022)
			} else {
				return
			}
			return

		case "loadFiles":
			//response.Response = Settings.Files

		// Daten schreiben
		case "saveSettings":
			ShowInfo("WEB:Saving settings")
			var authenticationUpdate = Settings.AuthenticationWEB
			var previousStoreBufferInRAM = Settings.StoreBufferInRAM
			var previousBindingIPs = Settings.BindingIPs
			var previousUseHttps = Settings.UseHttps
			response.Settings, err = updateServerSettings(request)
			if err == nil && response.Settings != nil {

				response.OpenMenu = strconv.Itoa(indexOfString("settings", System.WEB.Menu))

				if Settings.AuthenticationWEB && !authenticationUpdate {
					response.Reload = true
				}

				if Settings.StoreBufferInRAM != previousStoreBufferInRAM {
					streamManager.FileSystem = InitBufferVFS(Settings.StoreBufferInRAM)
				}

				if Settings.BindingIPs != previousBindingIPs || Settings.UseHttps != previousUseHttps {
					ShowInfo("WEB:Restart program since listening IP option has been changed!")
					err := webServer.restartWebserver()
					if err != nil {
						ShowError(err, 0)
					}
				}
			}

		case "saveFilesM3U":

			// Reset cache for urls.json
			var filename = getPlatformFile(System.Folder.Config + "urls.json")
			saveMapToJSONFile(filename, make(map[string]StreamInfo))
			Data.Cache.StreamingURLS = make(map[string]*StreamInfo)

			err = saveFiles(request, "m3u")
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("playlist", System.WEB.Menu))
			}
			updateUrlsJson()

		case "updateFileM3U":
			err = updateFile(request, "m3u")
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("playlist", System.WEB.Menu))
			}

		case "saveFilesHDHR":
			err = saveFiles(request, "hdhr")
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("playlist", System.WEB.Menu))
			}

		case "updateFileHDHR":
			err = updateFile(request, "hdhr")
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("playlist", System.WEB.Menu))
			}

		case "saveFilesXMLTV":
			err = saveFiles(request, "xmltv")
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("xmltv", System.WEB.Menu))
			}

		case "updateFileXMLTV":
			err = updateFile(request, "xmltv")
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("xmltv", System.WEB.Menu))
			}

		case "saveFilter":
			response.Settings, err = saveFilter(request)
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("filter", System.WEB.Menu))
			}

		case "saveEpgMapping":
			err = saveXEpgMapping(request)

		case "saveUserData":
			err = saveUserData(request)
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("users", System.WEB.Menu))
			}

		case "saveNewUser":
			err = saveNewUser(request)
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("users", System.WEB.Menu))
			}

		case "resetLogs":
			WebScreenLog.Log = make([]string, 0)
			WebScreenLog.Errors = 0
			WebScreenLog.Warnings = 0
			response.OpenMenu = strconv.Itoa(indexOfString("log", System.WEB.Menu))

		case "ThreadfinBackup":
			file, errNew := ThreadfinBackup()
			err = errNew
			if err == nil {
				response.OpenLink = fmt.Sprintf("%s://%s/download/%s", System.ServerProtocol, System.Domain, file)
			}

		case "ThreadfinRestore":
			WebScreenLog.Log = make([]string, 0)
			WebScreenLog.Errors = 0
			WebScreenLog.Warnings = 0

			if len(request.Base64) > 0 {

				newWebURL, err := ThreadfinRestoreFromWeb(request.Base64)
				if err != nil {
					ShowError(err, 000)
					response.Alert = err.Error()
				}

				if err == nil {

					if len(newWebURL) > 0 {
						response.Alert = "Backup was successfully restored.\nThe port of the sTeVe URL has changed, you have to restart Threadfin.\nAfter a restart, Threadfin can be reached again at the following URL:\n" + newWebURL
					} else {
						response.Alert = "Backup was successfully restored."
						response.Reload = true
					}
					ShowInfo("Threadfin:" + "Backup successfully restored.")
				}

			}

		case "uploadLogo":
			if len(request.Base64) > 0 {
				response.LogoURL, err = uploadLogo(request.Base64, request.Filename)

				if err == nil {

					if err = conn.WriteJSON(response); err != nil {
						ShowError(err, 1022)
					} else {
						return
					}

				}

			}

		case "uploadCustomImage":
			if len(request.Base64) > 0 {
				err = uploadCustomImage(request.Base64, request.Filename)
				if err != nil {
					ShowError(err, 1017)
				}
				ShowDebug("Sucessfully uploaded custom image", 1)
			}

		case "changeVersion":
			System.Beta = !System.Beta // Toggle Beta
			BinaryUpdate(true)

		case "updateThreadfin":
			BinaryUpdate(true)

		case "saveWizard":
			nextStep, errNew := saveWizard(request)

			err = errNew
			if err == nil {

				if nextStep == 10 {
					System.ConfigurationWizard = false
					response.Reload = true
				} else {
					response.Wizard = nextStep
				}

			}

			/*
				case "wizardCompleted":
					System.ConfigurationWizard = false
					response.Reload = true
			*/
		default:
			fmt.Println("+ + + + + + + + + + +", request.Cmd)

			var requestMap = make(map[string]interface{}) // Debug
			_ = requestMap
			if System.Dev {
				fmt.Println(mapToJSON(requestMap))
			}

		}

		if err != nil {
			response.Status = false
			response.Error = err.Error()
			response.Settings = &Settings
		}

		response = setDefaultResponseData(response, true)
		if System.ConfigurationWizard {
			response.ConfigurationWizard = System.ConfigurationWizard
		}

		if err = conn.WriteJSON(response); err != nil {
			ShowError(err, 1022)
		} else {
			break
		}

	}
}

func uploadCustomImage(input string, filename string) error {
	b64data := input[strings.IndexByte(input, ',')+1:]

	// decode Base64 string into bytes and save them to disk
	fileBytes, err := b64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return err
	}

	// get and delete former files in dir
	fileList, err := os.ReadDir(System.Folder.Custom)
	if err != nil {
		return err
	} else {
		for _, file := range fileList {
			os.Remove(System.Folder.Custom + file.Name())
		}
	}

	// delete generated video file if exist
	fileList, err = os.ReadDir(System.Folder.Video)
	if err != nil {
		return err
	} else {
		for _, file := range fileList {
			os.Remove(System.Folder.Custom + file.Name())
		}
	}

	// save file to disk
	var file = fmt.Sprintf("%s%s", System.Folder.Custom, filename)
	err = writeByteToFile(file, fileBytes)
	if err != nil {
		return err
	}
	return nil
}

// Web : Web Server /web/
func Web(w http.ResponseWriter, r *http.Request) {

	var lang = make(map[string]interface{})
	var err error

	var requestFile = strings.Replace(r.URL.Path, "/web", "web/public", 1)
	var webBasePath = "web/public/"
	var content, contentType, file string

	var language LanguageUI

	if System.Dev {

		lang, err = loadJSONFileToMap(fmt.Sprintf("%slang/%s.json", webBasePath, Settings.WebClientLanguage))
		if err != nil {
			ShowError(err, 000)
		}

	} else {

		var languageFile = fmt.Sprintf("%slang/%s.json", webBasePath, Settings.WebClientLanguage)

		if value, ok := webUI[languageFile].(string); ok {
			content = string(GetHTMLString(value))
			lang, err = jsonToMap(content)
		}
	}

	if err !=  nil {
		ShowError(err, 7001 )
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal([]byte(mapToJSON(lang)), &language)
	if err != nil {
		ShowError(err, 000)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	basePath := getFilenameFromPath(requestFile)

	if basePath == "public" {

		switch System.ConfigurationWizard {

		case true:
			file = requestFile + "configuration.html"
			Settings.AuthenticationWEB = false

		case false:
			file = requestFile + "index.html"

		}

		if System.ScanInProgress == 1 {
			file = requestFile + "maintenance.html"
		}

		switch Settings.AuthenticationWEB {
		case true:

			var username, password, confirm string
			switch r.Method {
			case "POST":
				var allUsers, _ = authentication.GetAllUserData()

				username = r.FormValue("username")
				password = r.FormValue("password")

				if len(allUsers) == 0 {
					confirm = r.FormValue("confirm")
				}

				// Erster Benutzer wird angelegt (Passwortbestätigung ist vorhanden)
				if len(confirm) > 0 {

					var token, err = createFirstUserForAuthentication(username, password)
					if err != nil {
						httpStatusError(w, http.StatusTooManyRequests)
						return
					}
					// Redirect, damit die Daten aus dem Browser gelöscht werden.
					w = authentication.SetCookieToken(w, token)
					http.Redirect(w, r, "/web", http.StatusMovedPermanently)
					return

				}

				// Benutzername und Passwort vorhanden, wird jetzt überprüft
				if len(username) > 0 && len(password) > 0 {

					var token, err = authentication.UserAuthentication(username, password)
					if err != nil {
						file = requestFile + "login.html"
						lang["authenticationErr"] = language.Login.Failed
						break
					}

					w = authentication.SetCookieToken(w, token)
					http.Redirect(w, r, "/web", http.StatusMovedPermanently) // Redirect, damit die Daten aus dem Browser gelöscht werden.

				} else {
					w = authentication.SetCookieToken(w, "-")
					http.Redirect(w, r, "/web", http.StatusMovedPermanently) // Redirect, damit die Daten aus dem Browser gelöscht werden.
				}

				return

			case "GET":
				lang["authenticationErr"] = ""
				_, token, err := authentication.CheckTheValidityOfTheTokenFromHTTPHeader(w, r)

				if err != nil {
					file = requestFile + "login.html"
					break
				}

				err = checkAuthorizationLevel(token, "authentication.web")
				if err != nil {
					file = requestFile + "login.html"
					break
				}

			}

			allUserData, err := authentication.GetAllUserData()
			if err != nil {
				ShowError(err, 000)
				httpStatusError(w, http.StatusForbidden)
				return
			}

			if len(allUserData) == 0 && Settings.AuthenticationWEB {
				file = requestFile + "create-first-user.html"
			}

		}

		requestFile = file

		if _, ok := webUI[requestFile]; ok {

			// content = GetHTMLString(value.(string))

			if contentType == "text/plain" {
				w.Header().Set("Content-Disposition", "attachment; filename="+getFilenameFromPath(requestFile))
			}

		} else {

			httpStatusError(w, http.StatusNotFound)
			return
		}

	}

	if value, ok := webUI[requestFile].(string); ok {

		content = string(GetHTMLString(value))
		contentType = getContentType(requestFile)

		if contentType == "text/plain" {
			w.Header().Set("Content-Disposition", "attachment; filename="+getFilenameFromPath(requestFile))
		}

	} else {
		httpStatusError(w, http.StatusNotFound)
		return
	}

	contentType = getContentType(requestFile)

	if System.Dev {
		// Lokale Webserver Dateien werden geladen, nur für die Entwicklung
		content, _ = readStringFromFile(requestFile)
	}

	w.Header().Add("Content-Type", contentType)
	w.WriteHeader(200)

	if contentType == "text/html" || contentType == "application/javascript" {
		if ContainsString(strings.Split(requestFile, "/"), "index.html") {
			content = strings.Replace(content, "en", Settings.WebClientLanguage, 1)
		}
		parsed_content, err := parseTemplate(content, lang)
		if err == nil {
			content = parsed_content
		}
	}

	w.Write([]byte(content))
}

// API : API request /api/
func API(w http.ResponseWriter, r *http.Request) {

	/*
			API Bedingungen (ohne Authentifizierung):
			- API muss in den Einstellungen aktiviert sein

			Beispiel API Request mit curl
			Status:
			curl -X POST -H "Content-Type: application/json" -d '{"cmd":"status"}' http://localhost:34400/api/

			- - - - -

			API Bedingungen (mit Authentifizierung):
			- API muss in den Einstellungen aktiviert sein
			- API muss bei den Authentifizierungseinstellungen aktiviert sein
			- Benutzer muss die Berechtigung API haben

			Nach jeder API Anfrage wird ein Token generiert, dieser ist einmal in 60 Minuten gültig.
			In jeder Antwort ist ein neuer Token enthalten

			Beispiel API Request mit curl
			Login:
			curl -X POST -H "Content-Type: application/json" -d '{"cmd":"login","username":"plex","password":"123"}' http://localhost:34400/api/

			Antwort:
			{
		  	"status": true,
		  	"token": "U0T-NTSaigh-RlbkqERsHvUpgvaaY2dyRGuwIIvv"
			}

			Status mit Verwendung eines Tokens:
			curl -X POST -H "Content-Type: application/json" -d '{"cmd":"status","token":"U0T-NTSaigh-RlbkqERsHvUpgvaaY2dyRGuwIIvv"}' http://localhost:4400/api/

			Antwort:
			{
			  "epg.source": "XEPG",
			  "status": true,
			  "streams.active": 7,
			  "streams.all": 63,
			  "streams.xepg": 2,
			  "token": "mXiG1NE1MrTXDtyh7PxRHK5z8iPI_LzxsQmY-LFn",
			  "url.dvr": "localhost:34400",
			  "url.m3u": "http://localhost:34400/m3u/threadfin.m3u",
			  "url.xepg": "http://localhost:34400/xmltv/threadfin.xml",
			  "version.api": "1.1.0",
			  "version.threadfin": "1.3.0"
			}
	*/

	var request APIRequestStruct
	var response APIResponseStruct

	responseAPIError := func(err error, statusCode int) {
		response.Error = err.Error()
		w.WriteHeader(statusCode)
		w.Write([]byte(mapToJSON(response)))
	}

	if !Settings.API {
		httpStatusError(w, http.StatusLocked)
		return
	}

	if r.Method == "GET" {
		httpStatusError(w, http.StatusNotFound)
		return
	}

	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		responseAPIError(err, http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(b, &request)
	if err != nil {
		responseAPIError(err, http.StatusBadRequest)
		return
	}

	w.Header().Set("content-type", "application/json")

	if Settings.AuthenticationAPI {
		token, err := handleAuthentication(request)
		if err != nil {
			responseAPIError(err, http.StatusUnauthorized)
			return
		}
		response.Token = token
	}

	switch request.Cmd {
	case "login":
		// No additional action needed
	case "status":
		populateStatusResponse(&response)
	case "getCurrentlyUsedChannels":
		err = GetCurrentlyUsedChannels(streamManager, &response)
		if err != nil {
			responseAPIError(err, http.StatusInternalServerError)
			return
		}
	case "updateM3U":
		err = updateM3U()
		if err != nil {
			responseAPIError(err, http.StatusInternalServerError)
			return
		}
	case "updateHDHR":
		err = updateHDHR()
		if err != nil {
			responseAPIError(err, http.StatusInternalServerError)
			return
		}
	case "updateXMLTV":
		err = updateXMLTV()
		if err != nil {
			responseAPIError(err, http.StatusInternalServerError)
			return
		}
	case "updateXEPG":
		buildXEPG(false)
	default:
		responseAPIError(errors.New(getErrMsg(5000)), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(mapToJSON(response)))
}

func handleAuthentication(request APIRequestStruct) (string, error) {
	var token string
	var err error

	if request.Token == "" {
		if request.Cmd == "login" {
			token, err = authentication.UserAuthentication(request.Username, request.Password)
		} else {
			err = errors.New("login incorrect")
		}
	} else {
		token, err = tokenAuthentication(request.Token)
	}

	if err != nil {
		return "", err
	}

	err = checkAuthorizationLevel(token, "authentication.api")
	if err != nil {
		return "", err
	}

	return token, nil
}

func populateStatusResponse(response *APIResponseStruct) {
	response.SystemInfo = populateSystemInfo()
}

func populateSystemInfo() *SystemInfoStruct {
	var systemInfo SystemInfoStruct
	systemInfo.APIVersion = System.APIVersion
	systemInfo.ThreadfinVersion = System.Version + "(" + System.Build + ")"
	systemInfo.EpgSource = Settings.EpgSource
	systemInfo.SystemURLs = populateSystemURLs()
	systemInfo.ChannelInfo = populateChannelInfo()
	return &systemInfo
}

func populateSystemURLs() SystemURLsStruct {
	var systemURLs SystemURLsStruct
	systemURLs.DVR = System.Domain
	systemURLs.M3U = System.ServerProtocol + "://" + System.Domain + "/m3u/threadfin.m3u"
	systemURLs.XEPG = System.ServerProtocol + "://" + System.Domain + "/xmltv/threadfin.xml"
	return systemURLs
}

func populateChannelInfo() ChannelInfoStruct {
	var channelInfo ChannelInfoStruct
	channelInfo.ActiveChannels = uint32(len(Data.Streams.Active))
	channelInfo.AllChannels = uint32(len(Data.Streams.All))
	channelInfo.XEPGChannels = uint32(Data.XEPG.XEPGCount)
	return channelInfo
}

func updateM3U() error {
	err := getProviderData("m3u", "")
	if err != nil {
		return err
	}
	return buildDatabaseDVR()
}

func updateHDHR() error {
	err := getProviderData("hdhr", "")
	if err != nil {
		return err
	}
	return buildDatabaseDVR()
}

func updateXMLTV() error {
	return getProviderData("xmltv", "")
}

// Download : Datei Download
func Download(w http.ResponseWriter, r *http.Request) {

	var path = r.URL.Path
	var file = System.Folder.Temp + getFilenameFromPath(path)
	w.Header().Set("Content-Disposition", "attachment; filename="+getFilenameFromPath(file))

	content, err := readStringFromFile(file)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	os.RemoveAll(System.Folder.Temp + getFilenameFromPath(path))
	w.Write([]byte(content))
}

func setDefaultResponseData(response ResponseStruct, data bool) (defaults ResponseStruct) {

	defaults = response

	// Folgende Daten immer an den Client übergeben
	defaults.ClientInfo.ARCH = System.ARCH
	defaults.ClientInfo.EpgSource = Settings.EpgSource
	defaults.ClientInfo.DVR = System.Addresses.DVR
	defaults.ClientInfo.M3U = System.Addresses.M3U
	defaults.ClientInfo.XML = System.Addresses.XML
	defaults.ClientInfo.OS = System.OS
	defaults.ClientInfo.Streams = fmt.Sprintf("%d / %d", len(Data.Streams.Active), len(Data.Streams.All))
	defaults.ClientInfo.UUID = Settings.UUID
	defaults.ClientInfo.Errors = WebScreenLog.Errors
	defaults.ClientInfo.Warnings = WebScreenLog.Warnings
	defaults.ClientInfo.SystemIPs = System.IPAddressesList
	defaults.Notification = System.Notification
	defaults.Log = WebScreenLog
	defaults.ClientInfo.Version = System.Version + " (" + System.Build + ")"
	defaults.ClientInfo.Beta = System.Beta

	if data {

		defaults.Users, _ = authentication.GetAllUserData()
		//defaults.DVR = System.DVRAddress

		if Settings.EpgSource == "XEPG" {

			defaults.ClientInfo.XEPGCount = Data.XEPG.XEPGCount

			var XEPG = make(map[string]interface{})

			if len(Data.Streams.Active) > 0 {

				XEPG["epgMapping"] = Data.XEPG.Channels
				XEPG["xmltvMap"] = Data.XMLTV.Mapping

			} else {

				XEPG["epgMapping"] = make(map[string]interface{})
				XEPG["xmltvMap"] = make(map[string]interface{})

			}

			defaults.XEPG = XEPG

		}

		defaults.Settings = &Settings

		defaults.Data.Playlist.M3U.Groups.Text = Data.Playlist.M3U.Groups.Text
		defaults.Data.Playlist.M3U.Groups.Value = Data.Playlist.M3U.Groups.Value
		defaults.Data.StreamPreviewUI.Active = Data.StreamPreviewUI.Active
		defaults.Data.StreamPreviewUI.Inactive = Data.StreamPreviewUI.Inactive

	}

	return
}

func enablePPV(w http.ResponseWriter, r *http.Request) {
	xepg, err := loadJSONFileToMap(System.File.XEPG)
	if err != nil {
		var response APIResponseStruct

		response.Error = err.Error()
		w.Write([]byte(mapToJSON(response)))
		w.WriteHeader(http.StatusInternalServerError)
	}

	for _, c := range xepg {

		var xepgChannel = c.(map[string]interface{})

		if xepgChannel["x-mapping"] == "PPV" {
			xepgChannel["x-active"] = true
		}
	}

	err = saveMapToJSONFile(System.File.XEPG, xepg)
	if err != nil {
		var response APIResponseStruct

		response.Error = err.Error()
		w.Write([]byte(mapToJSON(response)))
		w.WriteHeader(405)
		return
	}
	buildXEPG(false)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func disablePPV(w http.ResponseWriter, r *http.Request) {
	xepg, err := loadJSONFileToMap(System.File.XEPG)
	if err != nil {
		var response APIResponseStruct

		response.Error = err.Error()
		w.Write([]byte(mapToJSON(response)))
	}

	for _, c := range xepg {

		var xepgChannel = c.(map[string]interface{})

		if xepgChannel["x-mapping"] == "PPV" && xepgChannel["x-active"] == true {
			xepgChannel["x-active"] = false
		}
	}

	err = saveMapToJSONFile(System.File.XEPG, xepg)
	if err != nil {
		var response APIResponseStruct

		response.Error = err.Error()
		w.Write([]byte(mapToJSON(response)))
	}
	buildXEPG(false)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
}

func httpStatusError(w http.ResponseWriter, httpStatusCode int) {
	http.Error(w, fmt.Sprintf("%s [%d]", http.StatusText(httpStatusCode), httpStatusCode), httpStatusCode)
}

func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".ogg":
		return "video/ogg"
	case ".mp3":
		return "audio/mp3"
	case ".wav":
		return "audio/wav"
	default:
		return "text/plain"
	}
}
