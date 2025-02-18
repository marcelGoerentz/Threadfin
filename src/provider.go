package src

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	m3u "threadfin/src/internal/m3u-parser"
)

// fileType: Welcher Dateityp soll aktualisiert werden (m3u, hdhr, xml) | fileID: Update einer bestimmten Datei (Provider ID)
func getProviderData(fileType, fileID string) (err error) {

	var fileExtension, serverFileName string
	var body = make([]byte, 0)
	var newProvider = false
	var dataMap = make(map[string]interface{})

	var saveDateFromProvider = func(fileSource, serverFileName, id string, body []byte) (err error) {

		var data = make(map[string]interface{})

		if value, ok := dataMap[id].(map[string]interface{}); ok {
			data = value
		} else {
			data["id.provider"] = id
			dataMap[id] = data
		}

		// Default keys für die Providerdaten
		var keys = []string{"name", "description", "type", "file." + System.AppName, "file.source", "tuner", "http_proxy.ip", "http_proxy.port", "last.update", "compatibility", "counter.error", "counter.download", "provider.availability"}

		for _, key := range keys {

			if _, ok := data[key]; !ok {

				switch key {

				case "name":
					data[key] = serverFileName

				case "description":
					data[key] = ""

				case "type":
					data[key] = fileType

				case "file." + System.AppName:
					data[key] = id + fileExtension

				case "file.source":
					data[key] = fileSource

				case "last.update":
					data[key] = time.Now().Format("2006-01-02 15:04:05")

				case "tuner":
					if fileType == "m3u" || fileType == "hdhr" {
						if _, ok := data[key].(float64); !ok {
							data[key] = 1
						}
					}

				case "http_proxy.ip":
					data[key] = ""

				case "http_proxy.port":
					data[key] = ""

				case "compatibility":
					data[key] = make(map[string]interface{})

				case "counter.download":
					data[key] = 0.0

				case "counter.error":
					data[key] = 0.0

				case "provider.availability":
					data[key] = 100
				}

			}

		}

		if _, ok := data["id.provider"]; !ok {
			data["id.provider"] = id
		}

		// Datei extrahieren
		body, err = extractGZIP(body, fileSource)
		if err != nil {
			ShowError(err, 000)
			return
		}

		// Daten überprüfen
		ShowInfo("Check File:" + fileSource)
		switch fileType {

		case "m3u":
			_, err = m3u.MakeInterfaceFromM3U(body)

		case "hdhr":
			_, err = jsonToInterface(string(body))

		case "xmltv":
			err = checkXMLCompatibility(id, body)

		}

		if err != nil {
			return
		}

		var filePath = System.Folder.Data + data["file."+System.AppName].(string)

		err = writeByteToFile(filePath, body)

		if err == nil {
			data["last.update"] = time.Now().Format("2006-01-02 15:04:05")
			data["counter.download"] = data["counter.download"].(float64) + 1
		}

		return

	}

	switch fileType {

	case "m3u":
		dataMap = Settings.Files.M3U
		fileExtension = ".m3u"

	case "hdhr":
		dataMap = Settings.Files.HDHR
		fileExtension = ".json"

	case "xmltv":
		dataMap = Settings.Files.XMLTV
		fileExtension = ".xml"

	}

	for dataID, d := range dataMap {

		var data = d.(map[string]interface{})
		var fileSource = data["file.source"].(string)

		var httpProxyIp = ""
		if data["http_proxy.ip"] != nil {
			httpProxyIp = data["http_proxy.ip"].(string)
		}
		var httpProxyPort = ""
		if data["http_proxy.port"] != nil {
			httpProxyPort = data["http_proxy.port"].(string)
		}
		var httpProxyUrl = ""

		if httpProxyIp != "" && httpProxyPort != "" {
			httpProxyUrl = fmt.Sprintf("http://%s:%s", httpProxyIp, httpProxyPort)
		}

		newProvider = false

		if _, ok := data["new"]; ok {
			newProvider = true
			delete(data, "new")
		}

		// Wenn eine ID vorhanden ist und nicht mit der aus der Datanbank übereinstimmt, wird die Aktualisierung übersprungen (goto)
		if len(fileID) > 0 && !newProvider {
			if dataID != fileID {
				goto Done
			}
		}

		switch fileType {

		case "hdhr":

			// Laden vom HDHomeRun Tuner
			ShowInfo("Tuner:" + fileSource)
			var tunerURL = "http://" + fileSource + "/lineup.json"
			serverFileName, body, err = downloadFileFromServer(tunerURL, httpProxyUrl)

		default:

			if strings.Contains(fileSource, "http://") || strings.Contains(fileSource, "https://") {

				// Laden vom Remote Server
				ShowInfo("Download:" + fileSource)
				serverFileName, body, err = downloadFileFromServer(fileSource, httpProxyUrl)

			} else {

				// Laden einer lokalen Datei
				ShowInfo("Open:" + fileSource)

				err = checkFile(fileSource)
				if err == nil {
					body, err = readByteFromFile(fileSource)
					serverFileName = getFilenameFromPath(fileSource)
				}

			}

		}

		if err == nil {

			err = saveDateFromProvider(fileSource, serverFileName, dataID, body)
			if err == nil {
				ShowInfo("Save File:" + fileSource + " [ID: " + dataID + "]")
			}

		}

		if err != nil {

			ShowError(err, 000)
			var downloadErr = err

			if !newProvider {

				// Prüfen ob ältere Datei vorhanden ist
				var file = System.Folder.Data + dataID + fileExtension

				err = checkFile(file)
				if err == nil {

					if len(fileID) == 0 {
						ShowWarning(1011)
					}

					err = downloadErr
				}

				// Fehler Counter um 1 erhöhen
				var data = make(map[string]interface{})
				if value, ok := dataMap[dataID].(map[string]interface{}); ok {

					data = value
					data["counter.error"] = data["counter.error"].(float64) + 1
					data["counter.download"] = data["counter.download"].(float64) + 1

				}

			} else {
				return downloadErr
			}

		}

		// Berechnen der Fehlerquote
		if !newProvider {

			if value, ok := dataMap[dataID].(map[string]interface{}); ok {

				var data = make(map[string]interface{})
				data = value

				if data["counter.error"].(float64) == 0 {
					data["provider.availability"] = 100
				} else {
					data["provider.availability"] = int(data["counter.error"].(float64)*100/data["counter.download"].(float64)*-1 + 100)
				}

			}

		}

		switch fileType {

		case "m3u":
			Settings.Files.M3U = dataMap

		case "hdhr":
			Settings.Files.HDHR = dataMap

		case "xmltv":
			Settings.Files.XMLTV = dataMap
			delete(Data.Cache.XMLTV, System.Folder.Data+dataID+fileExtension)

		}

		saveSettings(Settings)

	Done:
	}

	return
}

func downloadFileFromServer(providerURL string, proxyUrl string) (filename string, body []byte, err error) {
	if proxyUrl != "" {
		ShowInfo("PROXY URL: " + proxyUrl)
	}

	_, err = url.ParseRequestURI(providerURL)
	if err != nil {
		return
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	if proxyUrl != "" {
		proxyURL, err := url.Parse(proxyUrl)
		if err != nil {
			return "", nil, err
		}

		httpClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
	}

	// Create a Request to set headers
	req, err := http.NewRequest("GET", providerURL, nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", Settings.UserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip,deflate")


	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("%d: %s %s", resp.StatusCode, providerURL, http.StatusText(resp.StatusCode))
		return
	}

	// Dateiname aus dem Header holen
	var index = strings.Index(resp.Header.Get("Content-Disposition"), "filename")

	if index > -1 {

		var headerFilename = resp.Header.Get("Content-Disposition")[index:]
		var value = strings.Split(headerFilename, `=`)
		var f = strings.Replace(value[1], `"`, "", -1)

		f = strings.Replace(f, `;`, "", -1)
		filename = f
		ShowInfo("Header filename:" + filename)

	} else {

		var cleanFilename = strings.SplitN(getFilenameFromPath(providerURL), "?", 2)
		filename = cleanFilename[0]

	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return
}
