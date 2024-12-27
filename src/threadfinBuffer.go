package src

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func StartThreadfinBuffer(stream *Stream, useBackup bool, backupNumber int, errorChan chan ErrorInfo) (*Buffer, error) {
	stopChan := make(chan struct{})
	ShowInfo(fmt.Sprintf("Streaming:Buffer:%s", "Threadfin"))
	ShowInfo("Streaming URL:" + stream.URL)

	go func() {
		resp, err := http.Get(stream.URL)
		if err != nil {
			return
		}
		//defer resp.Body.Close()
		if contentType, exists := resp.Header["Content-Type"]; exists {
			ShowDebug(fmt.Sprintf("Streaming:%s", contentType), 1)
			if contentType[0] != "application/octet-stream" {
				videoURL, audioURL, err := selectStreamFromMaster(resp.Body)
				if err != nil {
					ShowError(err, 0)
					return
				}

				if videoURL != "" || audioURL != "" {
					ShowInfo("Streaming: Can not stream from m3u file")
					errorChan <- ErrorInfo{4017, stream, ""}
					return
				}
			}
		}

		go HandleByteOutput(resp.Body, stream, errorChan)

		for {
			select {
			case <-stopChan:
				resp.Body.Close()
				time.Sleep(200 * time.Millisecond) // Let the buffer stop before going on
				return
			default:
				continue
			}
		}
	}()
	return &Buffer{isThirdPartyBuffer: false, stopChan: stopChan}, nil
}

func selectStreamFromMaster(resp io.ReadCloser) (string, string, error) {
	defer resp.Close()

	scanner := bufio.NewScanner(resp)
	var videoURL, audioURL string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#EXT-X-STREAM-INF") {
			scanner.Scan()
			videoURL = scanner.Text()
		}
		if strings.HasPrefix(line, "#EXT-X-MEDIA:TYPE=AUDIO") && strings.Contains(line, "DEFAULT=YES") {
			parts := strings.Split(line, "URI=\"")
			if len(parts) > 1 {
				audioURL = strings.Split(parts[1], "\"")[0]
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	return videoURL, audioURL, nil
}
