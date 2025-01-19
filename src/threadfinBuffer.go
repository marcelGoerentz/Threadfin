package src

import (
	"bufio"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

type ThreadfinBuffer struct {
	Stream *Stream
	StopChan chan struct{}
	VideoURL string
	AudioURL string
	PipeWriter *io.PipeWriter
}

func StartThreadfinBuffer(stream *Stream) error {
    stopChan := make(chan struct{})
    ShowInfo(fmt.Sprintf("Streaming:Buffer:%s", "Threadfin"))
    ShowInfo("Streaming URL:" + stream.URL)

    go func() {
        resp, err := http.Get(stream.URL)
        if err != nil {
            return
        }
        defer resp.Body.Close()

		var reader io.ReadCloser
		reader = resp.Body

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
						pr, pw := io.Pipe()
						threadfinBuf := &ThreadfinBuffer{
							Stream: stream,
							StopChan: stopChan,
							VideoURL:  videoURL,
							AudioURL: audioURL,
							PipeWriter: pw,
						}
                        go getTsSegments(threadfinBuf)
						reader = pr
                    }
                }
            }
        }

        go stream.Buffer.HandleByteOutput(reader) // Download the video file directly and save to disk

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

func getTsSegments(threadfinBuf *ThreadfinBuffer) {
    videoUrlBase := getUrlBase(threadfinBuf.VideoURL)
    audioUrlBase := getUrlBase(threadfinBuf.AudioURL)
    audioData := make([]byte, 1024*1024)
    segment := 1

    for {
        select {
        case <-threadfinBuf.StopChan:
            return
        default:
            data, err := DownloadElement(threadfinBuf.VideoURL)
            if err != nil {
                threadfinBuf.Stream.ReportError(err, 4017, "", false)
                return
            }
            videoSegments := ExtractSegments(string(data))
            data, err = DownloadElement(threadfinBuf.AudioURL)
            if err != nil {
                threadfinBuf.Stream.ReportError(err, 4017, "", false)
                return
            }
            audioSegments := ExtractSegments(string(data))
            videoSegments, audioSegments = synchronizeFiles(videoSegments, audioSegments)
            for i, name := range videoSegments {
                videoData, err := DownloadElement(videoUrlBase + name)
                if err != nil {
                    threadfinBuf.Stream.ReportError(err, 4017, "", false)
                    return
                }

                if i < len(audioSegments) {
                    audioName := audioSegments[i]
                    audioData, err = DownloadElement(audioUrlBase + audioName)
                    if err != nil {
                        threadfinBuf.Stream.ReportError(err, 4017, "", false)
                        return
                    }
                }


				if audioData != nil && videoData != nil {
                    err := threadfinBuf.combineAndSaveToBuffer(videoData, audioData, segment)
                    if err != nil {
                        threadfinBuf.Stream.ReportError(err, 4017, "", false)
                        return
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

func (tb *ThreadfinBuffer) combineAndSaveToBuffer(videoData []byte, audioData []byte, segment int) error {
	tmpFolder := tb.Stream.Folder + string(os.PathSeparator) + "tmp" + string(os.PathSeparator)
    videoFile := fmt.Sprintf("%svideo_segment_%d.ts", tmpFolder, segment)
    audioFile := fmt.Sprintf("%ssaudio_segment_%d.aac", tmpFolder, segment)

	_, err := os.Stat(tmpFolder)

	if fsIsNotExistErr(err) {
		os.MkdirAll(tmpFolder, 0755)
	}

    // Write video data to file
    if err := os.WriteFile(videoFile, videoData, 0644); err != nil {
        return err
    }

    // Write audio data to file
    if err := os.WriteFile(audioFile, audioData, 0644); err != nil {
        return err
    }

    // Combine video and audio using ffmpeg
    cmd := exec.Command(Settings.FFmpegPath, "-i", videoFile, "-i", audioFile, "-c", "copy", "-f", "hls", "pipe:1")
	cmd.Stdout = tb.PipeWriter
    if err := cmd.Run(); err != nil {
        return err
    }

    // Clean up temporary files
	os.Remove(videoFile)
   	os.Remove(audioFile)

    return nil
}