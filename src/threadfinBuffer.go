package src

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/asticode/go-astits"
)

func StartThreadfinBuffer(stream *Stream) error {
	stopChan := make(chan struct{})
	ShowInfo(fmt.Sprintf("Streaming:Buffer:%s", "Threadfin"))
	ShowInfo("Streaming URL:" + stream.URL)

	go func() {
		var readCloser io.ReadCloser
		resp, err := http.Get(stream.URL)
		if err != nil {
			return
		}
		readCloser = resp.Body
		
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
						readCloser, err = getTsSegments(stream, videoURL, audioURL)
						if err != nil {
							return
						}
					}
				}
			}
		}

		go stream.Buffer.HandleByteOutput(readCloser) // Download the video file directly and save to disk

		for {
			select {
			case <-stopChan:
				readCloser.Close()
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

func ExtractSegments(playlist string) []string {
	var segments []string
	lines := strings.Split(playlist, "\n")
	for _, line := range lines {
		if strings.HasSuffix(line, ".ts") || strings.HasSuffix(line, ".aac") {
			segments = append(segments, line)
		}
	}
	return segments
}

func DownloadElement(URL string) ([]byte, error) {
	response, err := http.Get(URL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func getTsSegments(stream *Stream, videoURL string, audioURL string) (io.ReadCloser, error) {
	videoUrlBase := getUrlBase(videoURL)
	audioUrlBase := getUrlBase(audioURL)
	audioData := make([]byte, 1024*1024)
	segment := 1
	var buffer bytes.Buffer

	for {
		select {
		case <-stream.Buffer.StopChan:
			return io.NopCloser(&buffer), nil
		default:
			data, err := DownloadElement(videoURL)
			if err != nil {
				stream.ReportError(err, 4017, "", false)
				return nil, err
			}
			videoSegments := ExtractSegments(string(data))
			data, err = DownloadElement(audioURL)
			if err != nil {
				stream.ReportError(err, 4017, "", false)
				return nil, err
			}
			audioSegments := ExtractSegments(string(data))
			videoSegments, audioSegments = synchronizeFiles(videoSegments, audioSegments)
			for i, name := range videoSegments {
				videoData, err := DownloadElement(videoUrlBase + name)
				if err != nil {
					stream.ReportError(err, 4017, "", false)
					return nil, err
				}

				if i < len(audioSegments) {
					audioName := audioSegments[i]
					audioData, err = DownloadElement(audioUrlBase + audioName)
					if err != nil {
						stream.ReportError(err, 4017, "", false)
						return nil, err
					}
				}

				if audioData != nil && videoData != nil {
					err := combineAndSendToBuffer(&buffer, videoData, audioData)
					if err != nil {
						stream.ReportError(err, 4017, "", false)
						return nil, err
					}
					segment++
				}
			}
		}
	}
}

func getUrlBase(URL string) string {
	u, err := url.Parse(URL)
	if err != nil {
		return ""
	}
	parts := strings.Split(u.Path, "/")
	path := strings.Join(parts[:len(parts)-1], "/")
	return u.Scheme + "://" + u.Host + path + "/"
}

func synchronizeFiles(videoSegments []string, audioSegments []string) ([]string, []string) {
	for i := range videoSegments {
		for j := range audioSegments {
			if strings.Split(videoSegments[i], ".")[0] == strings.Split(audioSegments[j], ".")[0] {
				return videoSegments[i:], audioSegments[j:]
			}
		}
	}
	return videoSegments, audioSegments
}

func combineAndSendToBuffer(buffer *bytes.Buffer, videoData []byte, audioData []byte) error {
	videoPackets, err := extractVideoData(videoData, 256)
	if err != nil {
		return err
	}
	audioPackets, err := readAudioData(audioData)
	if err != nil {
		return err
	}

	w := astits.NewMuxer(context.Background(), buffer)
	for _, packet := range videoPackets {
		if _, err := w.WriteData(&astits.MuxerData{PID: 256, PES: &astits.PESData{Data: packet}}); err != nil {
			return err
		}
	}
	for _, packet := range audioPackets {
		if _, err := w.WriteData(&astits.MuxerData{PID: 257, PES: &astits.PESData{Data: packet}}); err != nil {
			return err
		}
	}

	return nil
}

func extractVideoData(tsData []byte, pid uint16) ([][]byte, error) {
	var packets [][]byte
	r := astits.NewDemuxer(context.Background(), bytes.NewReader(tsData))
	for {
		d, err := r.NextData()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if d.PID == pid {
			packets = append(packets, d.PES.Data)
		}
	}
	return packets, nil
}

func readAudioData(audioData []byte) ([][]byte, error) {
	var packets [][]byte
	r := astits.NewDemuxer(context.Background(), bytes.NewReader(audioData))
	for {
		d, err := r.NextData()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if d.PID == 257 {
			packets = append(packets, d.PES.Data)
		}
	}
	return packets, nil
}
