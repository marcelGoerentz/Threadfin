package src

import (
	"bufio"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

func StartThreadfinBuffer(stream *Stream, errorChan chan ErrorInfo) error {
	stopChan := make(chan struct{})
	ShowInfo(fmt.Sprintf("Streaming:Buffer:%s", "Threadfin"))
	ShowInfo("Streaming URL:" + stream.URL)

	go func() {
		resp, err := http.Get(stream.URL)
		if err != nil {
			return
		}
		
		if contentTypes, exists := resp.Header["Content-Type"]; exists {
			ShowDebug(fmt.Sprintf("Streaming:%s", contentTypes), 1)
			extensions := []string{}
			for _, contentType := range contentTypes {
				type_extensions, err := mime.ExtensionsByType(contentType)
				if err != nil {
					return
				}
				if type_extensions == nil {
					continue
				}
				extensions = append(extensions, type_extensions...)
			}
						
			for _, extension := range extensions {
				if extension == ".m3u" || extension == ".m3u8" {
					videoURL, audioURL, err := selectStreamFromMaster(resp.Body)
					if err != nil {
						ShowError(err, 0)
						return
					}

					if videoURL != "" || audioURL != "" {
						ShowInfo("Streaming:Can not stream from m3u file")
						errorChan <- ErrorInfo{4017, stream, ""}
						return
					}
				}
			}
		}

		go HandleByteOutput(resp.Body, stream, errorChan) // Download the video file directly and save to disk

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
	stream.Buffer.StopChan = stopChan
	return nil
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
