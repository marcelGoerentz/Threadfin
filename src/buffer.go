package src

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	//"net/url"
	"os"
	"os/exec"

	//"path"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/osfs"
)

var activeClientCount int
var activePlaylistCount int

func getActiveClientCount() (count int) {
	return activeClientCount
}

func getActivePlaylistCount() (count int) {
	return activePlaylistCount
}

func createStreamID(stream map[int]ThisStream) int {
    for i := 0; ; i++ {
        if _, ok := stream[i]; !ok {
            debug := fmt.Sprintf("Streaming Status:Stream ID = %d", i)
            showDebug(debug, 1)
            return i
        }
	}
}

func createAlternativNoMoreStreamsVideo(pathToFile string) (error) {
	cmd := new(exec.Cmd)
	switch Settings.Buffer {
	case "ffmpeg":
		cmd = exec.Command(Settings.FFmpegPath, "-loop", "1", "-i", pathToFile, "-c:v", "libx264", "-t", "1", "-pix_fmt", "yuv420p", "-vf", "scale=1920:1080", fmt.Sprintf("%sstream-limit.ts", System.Folder.Video))
	case "vlc":
		cmd = exec.Command(Settings.VLCPath, "--no-audio", "--loop", "--sout", fmt.Sprintf("'#transcode{vcodec=h264,vb=1024,scale=1,width=1920,height=1080,acodec=none,venc=x264{preset=ultrafast}}:standard{access=file,mux=ts,dst=%sstream-limit.ts}'", System.Folder.Video), System.Folder.Video, pathToFile)

	}
	if len(cmd.Args) > 0 {
		showInfo("Streaming Status:Creating video from uploaded image for a customized no more stream video")
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func createPlaylist(streamInfo StreamInfo) Playlist {
	var playlist Playlist
	playlist.Folder = System.Folder.Temp + streamInfo.PlaylistID + string(os.PathSeparator)
	playlist.PlaylistID = streamInfo.PlaylistID
	playlist.Streams = make(map[int]ThisStream)
	playlist.Clients = make(map[int]ThisClient)
	return playlist
}

func createNewStream(streamInfo StreamInfo) ThisStream{
	var stream ThisStream
	stream.URL = streamInfo.URL
	stream.BackupChannel1URL = streamInfo.BackupChannel1URL
	stream.BackupChannel2URL = streamInfo.BackupChannel2URL
	stream.BackupChannel3URL = streamInfo.BackupChannel3URL
	stream.ChannelName = streamInfo.Name
	stream.Status = false
	return stream
}

func bufferingStream(streamInfo StreamInfo, w http.ResponseWriter, r *http.Request) {

	time.Sleep(time.Duration(Settings.BufferTimeout) * time.Millisecond)

	var playlist Playlist
	var client ThisClient
	var stream ThisStream
	var streaming = false
	var streamID int
	var debug string
	var timeOut = 0
	var newStream = true

	//w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Connection", "close")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Überprüfen ob die Playlist schon verwendet wird
	if p, ok := BufferInformation.Load(streamInfo.PlaylistID); !ok {

		var playlistType string

		// Playlist wird noch nicht verwendet, Default-Werte für die Playlist erstellen
		playlist = createPlaylist(streamInfo)

		err := checkVFSFolder(playlist.Folder, bufferVFS)
		if err != nil {
			ShowError(err, 000)
			httpStatusError(w, http.StatusNotFound)
			return
		}

		switch playlist.PlaylistID[0:1] {

		case "M":
			playlistType = "m3u"

		case "H":
			playlistType = "hdhr"

		}

		playlist.Tuner = getTuner(playlist.PlaylistID, playlistType)

		playlist.PlaylistName = getProviderParameter(playlist.PlaylistID, playlistType, "name")
		playlist.HttpProxyIP = getProviderParameter(playlist.PlaylistID, playlistType, "http_proxy.ip")
		playlist.HttpProxyPort = getProviderParameter(playlist.PlaylistID, playlistType, "http_proxy.port")

		// Default-Werte für den Stream erstellen
		streamID = createStreamID(playlist.Streams)

		client.Connection = 1
		activeClientCount += 1
		activePlaylistCount += 1

		stream = createNewStream(streamInfo)

		playlist.Streams[streamID] = stream
		playlist.Clients[streamID] = client

		BufferInformation.Store(playlist.PlaylistID, playlist)

	} else {

		// Playlist wird bereits zum streamen verwendet
		// Überprüfen ob die URL bereits von einem anderen Client gestreamt wird.

		playlist = p.(Playlist)

		for id := range playlist.Streams {

			stream = playlist.Streams[id]
			client = playlist.Clients[id]

			if streamInfo.URL == stream.URL {

				streamID = id
				newStream = false
				client.Connection += 1
				activeClientCount += 1

				playlist.Streams[streamID] = stream
				playlist.Clients[streamID] = client

				BufferInformation.Store(playlist.PlaylistID, playlist)

				debug = fmt.Sprintf("Restream Status:Playlist: %s - Channel: %s - Connections: %d", playlist.PlaylistName, stream.ChannelName, client.Connection)

				showDebug(debug, 1)

				if c, ok := BufferClients.Load(playlist.PlaylistID + stream.MD5); ok {

					var clients = c.(ClientConnection)
					clients.Connection = clients.Connection + 1
					log.Println("CLIENTS: ", clients)

					BufferClients.Store(playlist.PlaylistID + stream.MD5, clients)

				}

				break
			}

		}

		// Neuer Stream bei einer bereits aktiven Playlist
		if newStream {

			// Prüfen ob die Playlist noch einen weiteren Stream erlaubt (Tuner)
			if len(playlist.Streams) >= playlist.Tuner {

				showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - No new connections available. Tuner = %d", playlist.PlaylistName, playlist.Tuner))
				var content []byte
				var contentOk, customizedVideo bool

				imageFileList, err := os.ReadDir(System.Folder.Custom)
				if err != nil {
					ShowError(err, 0)
				}
				// Check if a customized video is available and use it if so
				fileList, err := os.ReadDir(System.Folder.Video)
				if err == nil {
					var createContent = false
					switch len(fileList) {
					case 0: // If no customized video is available create one
						createContent = true
					case 1: // Is there only one file, use it
						break
					default:
						// remove all found files
						for _, file := range fileList {
							os.Remove(System.Folder.Video + file.Name())
						}
						createContent = true
					}
					if createContent {
						if len(imageFileList) > 0 {
							err := createAlternativNoMoreStreamsVideo(System.Folder.Custom + imageFileList[0].Name())
							if err == nil {
								contentOk = true
								customizedVideo = true
							} else {
								ShowError(err, 0) // log error
								return
							}
						}
					}
				}

				if value, ok := webUI["html/video/stream-limit.ts"]; ok && !customizedVideo {

					contentOk = true
					content = GetHTMLString(value.(string))
				}

				if contentOk {
					w.Header().Set("Content-type", "video/mpeg")
					w.Header().Set("Content-Length:", fmt.Sprintf("%d", len(content)))
					w.WriteHeader(http.StatusOK)

					for i := 0; i < 60; i++ {
						if _, err := w.Write(content); err != nil {
							ShowError(err, 0) // log error
							return
						}
						time.Sleep(500 * time.Millisecond)
					}
				}
				return
			}

			// Playlist erlaubt einen weiterern Stream (Das Limit des Tuners ist noch nicht erreicht)
			// Default-Werte für den Stream erstellen
			stream = ThisStream{}
			client = ThisClient{}

			streamID = createStreamID(playlist.Streams)

			client.Connection += 1
			activePlaylistCount += 1
			stream = createNewStream(streamInfo)

			playlist.Streams[streamID] = stream
			playlist.Clients[streamID] = client

			BufferInformation.Store(playlist.PlaylistID, playlist)

		}
	}

	// Überprüfen ob der Stream breits von einem anderen Client abgespielt wird
	if !playlist.Streams[streamID].Status && newStream {

		// Neuer Buffer wird benötigt
		stream = playlist.Streams[streamID]
		stream.MD5 = getMD5(streamInfo.URL)
		stream.Folder = playlist.Folder + stream.MD5 + string(os.PathSeparator)
		stream.PlaylistID = playlist.PlaylistID
		stream.PlaylistName = playlist.PlaylistName

		playlist.Streams[streamID] = stream
		BufferInformation.Store(playlist.PlaylistID, playlist)

		switch Settings.Buffer {

		case "ffmpeg", "vlc":
			go thirdPartyBuffer(streamID, playlist.PlaylistID, false, 0)

		default:
			break

		}

		showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))

		var clients ClientConnection
		clients.Connection = 1
		BufferClients.Store(playlist.PlaylistID + stream.MD5, clients)

	}

	w.WriteHeader(200)

	for { // Loop 1: Warten bis das erste Segment durch den Buffer heruntergeladen wurde

		if p, ok := BufferInformation.Load(playlist.PlaylistID); ok {

			var playlist = p.(Playlist)

			if stream, ok := playlist.Streams[streamID]; ok {

				if !stream.Status {

					timeOut++

					time.Sleep(time.Duration(100) * time.Millisecond)

					if c, ok := BufferClients.Load(playlist.PlaylistID + stream.MD5); ok {

						var clients = c.(ClientConnection)

						if clients.Error != nil || (timeOut > 200 && (playlist.Streams[streamID].BackupChannel1URL == "" && playlist.Streams[streamID].BackupChannel2URL == "" && playlist.Streams[streamID].BackupChannel3URL == "")) {
							killClientConnection(streamID, stream.PlaylistID, false)
							return
						}

					}

					continue
				}

				var oldSegments []string

				for { // Loop 2: Temporäre Datein sind vorhanden, Daten können zum Client gesendet werden

					// HTTP Clientverbindung überwachen

					ctx := r.Context()
					if ok {

						select {

						case <-ctx.Done():
							killClientConnection(streamID, playlist.PlaylistID, false)
							return

						default:
							if c, ok := BufferClients.Load(playlist.PlaylistID + stream.MD5); ok {

								var clients = c.(ClientConnection)
								if clients.Error != nil {
									ShowError(clients.Error, 0)
									killClientConnection(streamID, playlist.PlaylistID, false)
									return
								}

							} else {

								return

							}

						}

					}

					if _, err := bufferVFS.Stat(stream.Folder); fsIsNotExistErr(err) {
						killClientConnection(streamID, playlist.PlaylistID, false)
						return
					}

					var tmpFiles = getBufTmpFiles(&stream)
					//fmt.Println("Buffer Loop:", stream.Connection)

					for _, f := range tmpFiles {

						if _, err := bufferVFS.Stat(stream.Folder); fsIsNotExistErr(err) {
							killClientConnection(streamID, playlist.PlaylistID, false)
							return
						}

						oldSegments = append(oldSegments, f)

						var fileName = stream.Folder + f

						file, err := bufferVFS.Open(fileName)
						if err != nil {
							debug = fmt.Sprintf("Buffer Open (%s)", fileName)
							showDebug(debug, 2)
							return
						}
						defer file.Close()

						if err == nil {

							l, err := file.Stat()
							if err == nil {

								debug = fmt.Sprintf("Buffer Status:Send to client (%s)", fileName)
								showDebug(debug, 2)

								var buffer = make([]byte, int(l.Size()))
								_, err = file.Read(buffer)

								if err == nil {

									file.Seek(0, 0)

									if !streaming {

										contentType := http.DetectContentType(buffer)
										_ = contentType
										//w.Header().Set("Content-type", "video/mpeg")
										w.Header().Set("Content-type", contentType)
										w.Header().Set("Content-Length", "0")
										w.Header().Set("Connection", "close")

									}

									/*
									   // HDHR Header
									   w.Header().Set("Cache-Control", "no-cache")
									   w.Header().Set("Pragma", "no-cache")
									   w.Header().Set("transferMode.dlna.org", "Streaming")
									*/

									_, err := w.Write(buffer)

									if err != nil {
										file.Close()
										killClientConnection(streamID, playlist.PlaylistID, false)
										return
									}

									file.Close()
									killClientConnection(streamID, playlist.PlaylistID, false)
									return
								}

								file.Close()

							}

							var n = indexOfString(f, oldSegments)

							if n > 20 {

								var fileToRemove = stream.Folder + oldSegments[0]
								if err = bufferVFS.RemoveAll(getPlatformFile(fileToRemove)); err != nil {
									ShowError(err, 4007)
								}
								oldSegments = append(oldSegments[:0], oldSegments[0+1:]...)

							}

						}

						file.Close()

					}

					if len(tmpFiles) == 0 {
						time.Sleep(time.Duration(100) * time.Millisecond)
					}

				} // Ende Loop 2

			} else {

				// Stream nicht vorhanden
				killClientConnection(streamID, stream.PlaylistID, false)
				showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))
				return

			}

		} // Ende BufferInformation

	} // Ende Loop 1

}

func getBufTmpFiles(stream *ThisStream) (tmpFiles []string) {

	var tmpFolder = stream.Folder
	var fileIDs []float64

	if _, err := bufferVFS.Stat(tmpFolder); !fsIsNotExistErr(err) {

		files, err := bufferVFS.ReadDir(getPlatformPath(tmpFolder))
		if err != nil {
			ShowError(err, 000)
			return
		}

		if len(files) > 2 {

			for _, file := range files {

				var fileID = strings.Replace(file.Name(), ".ts", "", -1)
				var f, err = strconv.ParseFloat(fileID, 64)

				if err == nil {
					fileIDs = append(fileIDs, f)
				}

			}

			sort.Float64s(fileIDs)
			fileIDs = fileIDs[:len(fileIDs)-1]

			for _, file := range fileIDs {

				var fileName = fmt.Sprintf("%d.ts", int64(file))

				if indexOfString(fileName, stream.OldSegments) == -1 {
					tmpFiles = append(tmpFiles, fileName)
					stream.OldSegments = append(stream.OldSegments, fileName)
				}

			}

		}

	}

	return
}

func killClientConnection(streamID int, playlistID string, force bool) {

	Lock.Lock()
	defer Lock.Unlock()

	if p, ok := BufferInformation.Load(playlistID); ok {

		var playlist = p.(Playlist)

		if force {
			delete(playlist.Streams, streamID)
			showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))
			return
		}

		if stream, ok := playlist.Streams[streamID]; ok {

			if c, ok := BufferClients.Load(playlistID + stream.MD5); ok {

				var clients = c.(ClientConnection)
				clients.Connection -= 1
				activeClientCount -= 1
				if activeClientCount <= 0 {
					activeClientCount = 0
				}
				BufferClients.Store(playlistID+stream.MD5, clients)

				showInfo("Streaming Status:Client has terminated the connection")
				showInfo(fmt.Sprintf("Streaming Status:Channel: %s (Clients: %d)", stream.ChannelName, clients.Connection))

				if clients.Connection <= 0 {
					if activePlaylistCount > 0 {
						activePlaylistCount -= 1
					} else {
						activePlaylistCount = 0
					}
					BufferClients.Delete(playlistID + stream.MD5)
					delete(playlist.Streams, streamID)
					delete(playlist.Clients, streamID)
				}

			}

			BufferInformation.Store(playlistID, playlist)

			if len(playlist.Streams) > 0 {
				showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))
			}

		}

	}

}

func clientConnection(stream ThisStream) (status bool) {

	status = true
	Lock.Lock()
	defer Lock.Unlock()

	if _, ok := BufferClients.Load(stream.PlaylistID + stream.MD5); !ok {

		var debug = fmt.Sprintf("Streaming Status:Remove temporary files (%s)", stream.Folder)
		showDebug(debug, 1)

		status = false

		debug = fmt.Sprintf("Remove tmp folder:%s", stream.Folder)
		showDebug(debug, 1)

		if err := bufferVFS.RemoveAll(stream.Folder); err != nil {
			ShowError(err, 4005)
		}

		if p, ok := BufferInformation.Load(stream.PlaylistID); ok {

			showInfo(fmt.Sprintf("Streaming Status:Channel: %s - No client is using this channel anymore. Streaming Server connection has ended", stream.ChannelName))

			var playlist = p.(Playlist)

			showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))

			if len(playlist.Streams) <= 0 {
				BufferInformation.Delete(stream.PlaylistID)
			}

		}

		status = false

	}

	return
}

// Buffer mit FFMPEG oder VLC
func thirdPartyBuffer(streamID int, playlistID string, useBackup bool, backupNumber int) {

	if p, ok := BufferInformation.Load(playlistID); ok {

		var playlist = p.(Playlist)
		var debug, path, options, bufferType string
		var tmpSegment = 1
		var bufferSize = Settings.BufferSize * 1024
		var stream = playlist.Streams[streamID]
		var buf bytes.Buffer
		var fileSize = 0
		var streamStatus = make(chan bool)

		var tmpFolder = playlist.Streams[streamID].Folder
		var url = playlist.Streams[streamID].URL
		if useBackup {
			if backupNumber >= 1 && backupNumber <= 3 {
				switch backupNumber {
				case 1:
					url = playlist.Streams[streamID].BackupChannel1URL
					showHighlight("START OF BACKUP 1 STREAM")
					showInfo("Backup Channel 1 URL: " + url)
				case 2:
					url = playlist.Streams[streamID].BackupChannel2URL
					showHighlight("START OF BACKUP 2 STREAM")
					showInfo("Backup Channel 2 URL: " + url)
				case 3:
					url = playlist.Streams[streamID].BackupChannel3URL
					showHighlight("START OF BACKUP 3 STREAM")
					showInfo("Backup Channel 3 URL: " + url)
				}
			}
		}

		stream.Status = false

		bufferType = strings.ToUpper(Settings.Buffer)

		switch Settings.Buffer {

		case "ffmpeg":
			path = Settings.FFmpegPath
			options = fmt.Sprintf("%s", Settings.FFmpegOptions)

		case "vlc":
			path = Settings.VLCPath
			options = fmt.Sprintf("%s meta-title=Threadfin", Settings.VLCOptions)

		default:
			return
		}

		var addErrorToStream = func(err error) {

			if !useBackup || (useBackup && backupNumber >= 0 && backupNumber <= 3) {
				backupNumber = backupNumber + 1
				if playlist.Streams[streamID].BackupChannel1URL != "" || playlist.Streams[streamID].BackupChannel2URL != "" || playlist.Streams[streamID].BackupChannel3URL != "" {
					thirdPartyBuffer(streamID, playlistID, true, backupNumber)
				}
			}

			var stream = playlist.Streams[streamID]

			if c, ok := BufferClients.Load(playlistID + stream.MD5); ok {

				var clients = c.(ClientConnection)
				clients.Error = err
				BufferClients.Store(playlistID+stream.MD5, clients)

			}

		}

		if err := bufferVFS.RemoveAll(getPlatformPath(tmpFolder)); err != nil {
			ShowError(err, 4005)
		}

		err := checkVFSFolder(tmpFolder, bufferVFS)
		if err != nil {
			ShowError(err, 0)
			addErrorToStream(err)
			return
		}

		err = checkFile(path)
		if err != nil {
			ShowError(err, 0)
			addErrorToStream(err)
			return
		}

		showInfo(fmt.Sprintf("%s path:%s", bufferType, path))
		showInfo("Streaming URL:" + url)

		var tmpFile = fmt.Sprintf("%s%d.ts", tmpFolder, tmpSegment)

		f, err := bufferVFS.Create(tmpFile)
		f.Close()
		if err != nil {
			addErrorToStream(err)
			return
		}

		//args = strings.Replace(args, "[USER-AGENT]", Settings.UserAgent, -1)

		// Argument list for the third party buffer
		var args []string

		for i, a := range strings.Split(options, " ") {

			switch bufferType {
			case "FFMPEG":
				a = strings.Replace(a, "[URL]", url, -1)
				if i == 0 {
					if len(Settings.UserAgent) != 0 {
						args = append(args, "-user_agent", Settings.UserAgent)
					}
					if playlist.HttpProxyIP != "" && playlist.HttpProxyPort != "" {
						args = append(args, "-http_proxy", fmt.Sprintf("http://%s:%s", playlist.HttpProxyIP, playlist.HttpProxyPort))
					}
					if playlist.HttpProxyIP != "" && playlist.HttpProxyPort != "" {
						args = append(args, "-http_proxy", fmt.Sprintf("http://%s:%s", playlist.HttpProxyIP, playlist.HttpProxyPort))
					}
				}

				args = append(args, a)

			case "VLC":
				if a == "[URL]" {
					a = strings.Replace(a, "[URL]", url, -1)
					args = append(args, a)

					if len(Settings.UserAgent) != 0 {
						args = append(args, fmt.Sprintf(":http-user-agent=%s", Settings.UserAgent))
					}

					if playlist.HttpProxyIP != "" && playlist.HttpProxyPort != "" {
						args = append(args, "-http_proxy", fmt.Sprintf("http://%s:%s", playlist.HttpProxyIP, playlist.HttpProxyPort))
					}

				} else {
					args = append(args, a)
				}

			}

		}

		var cmd = exec.Command(path, args...)
		//writePIDtoDisc(string(cmd.Process.Pid))

		debug = fmt.Sprintf("%s:%s %s", bufferType, path, args)
		showDebug(debug, 1)

		// Byte-Daten vom Prozess
		stdOut, err := cmd.StdoutPipe()
		if err != nil {
			ShowError(err, 0)
			terminateProcessGracefully(cmd)
			addErrorToStream(err)
			return
		}

		// Log-Daten vom Prozess
		logOut, err := cmd.StderrPipe()
		if err != nil {
			ShowError(err, 0)
			terminateProcessGracefully(cmd)
			addErrorToStream(err)
			return
		}

		if len(buf.Bytes()) == 0 && !stream.Status {
			showInfo(bufferType + ":Processing data")
		}

		cmd.Start()
		defer cmd.Wait()
		writePIDtoDisk(fmt.Sprintf("%d", cmd.Process.Pid))

		go func() {

			// Log Daten vom Prozess im Debug Mode 1 anzeigen.
			scanner := bufio.NewScanner(logOut)
			scanner.Split(bufio.ScanLines)

			for scanner.Scan() {

				debug = fmt.Sprintf("%s log:%s", bufferType, strings.TrimSpace(scanner.Text()))

				select {
				case <-streamStatus:
					showDebug(debug, 1)
				default:
					showInfo(debug)
				}

				time.Sleep(time.Duration(10) * time.Millisecond)

			}

		}()

		f, err = bufferVFS.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		buffer := make([]byte, 1024*4)

		reader := bufio.NewReader(stdOut)

		t := make(chan int)

		go func() {

			var timeout = 0
			for {
				time.Sleep(time.Duration(1000) * time.Millisecond)
				timeout++

				select {
				case <-t:
					return
				default:
					t <- timeout
				}

			}

		}()

		for {

			select {
			case timeout := <-t:
				if timeout >= 20 && tmpSegment == 1 {
					terminateProcessGracefully(cmd)
					err = errors.New("Timeout")
					ShowError(err, 4006)
					addErrorToStream(err)
					f.Close()
					return
				}

			default:

			}

			if fileSize == 0 && !stream.Status {
				showInfo("Streaming Status:Receive data from " + bufferType)
			}

			if !clientConnection(stream) {
				terminateProcessGracefully(cmd)
				f.Close()
				return
			}

			n, err := reader.Read(buffer)
			if err == io.EOF {
				break
			}

			fileSize = fileSize + len(buffer[:n])

			if _, err := f.Write(buffer[:n]); err != nil {
				terminateProcessGracefully(cmd)
				ShowError(err, 0)
				addErrorToStream(err)
				return
			}

			if fileSize >= bufferSize/2 {

				if tmpSegment == 1 && !stream.Status {
					close(t)
					close(streamStatus)
					showInfo(fmt.Sprintf("Streaming Status:Buffering data from %s", bufferType))
				}

				f.Close()
				tmpSegment++

				if !stream.Status {
					Lock.Lock()
					stream.Status = true
					playlist.Streams[streamID] = stream
					BufferInformation.Store(playlistID, playlist)
					Lock.Unlock()
				}

				tmpFile = fmt.Sprintf("%s%d.ts", tmpFolder, tmpSegment)

				fileSize = 0

				var errCreate, errOpen error
				_, errCreate = bufferVFS.Create(tmpFile)
				f, errOpen = bufferVFS.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0600)
				if errCreate != nil || errOpen != nil {
					terminateProcessGracefully(cmd)
					ShowError(err, 0)
					addErrorToStream(err)
					return
				}

			}

		}

		terminateProcessGracefully(cmd)

		err = errors.New(bufferType + " error")

		addErrorToStream(err)
		ShowError(err, 1204)

		time.Sleep(time.Duration(500) * time.Millisecond)
		clientConnection(stream)

		return
	}
}

func getTuner(id, playlistType string) (tuner int) {

	switch Settings.Buffer {

	case "-":
		tuner = Settings.Tuner

	case "threadfin", "ffmpeg", "vlc":

		i, err := strconv.Atoi(getProviderParameter(id, playlistType, "tuner"))
		if err == nil {
			tuner = i
		} else {
			ShowError(err, 0)
			tuner = 1
		}

	}

	return
}

func initBufferVFS(virtual bool) {

	if virtual {
		bufferVFS = memfs.New()
	} else {
		bufferVFS = osfs.New()
	}

}

func terminateProcessGracefully(cmd *exec.Cmd) {
	if cmd.Process != nil {
		cmd.Process.Signal(syscall.SIGKILL)
		cmd.Wait()
		deletPIDfromDisc(fmt.Sprintf("%d", cmd.Process.Pid))
	}
}

func writePIDtoDisk(pid string) {
	// Open the file in append mode (create it if it doesn't exist)
	file, err := os.OpenFile(System.Folder.Temp + "PIDs", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Write your text to the file
	_, err = file.WriteString(pid + "\n")
	if err != nil {
		log.Fatal(err)
	}
}

func deletPIDfromDisc(delete_pid string) (error) {
	file, err := os.OpenFile(System.Folder.Temp + "PIDs", os.O_RDWR, 0660)
	if err != nil {
		return err
	}
	// Create a scanner
	scanner := bufio.NewScanner(file)

	// Read line by line
	pids := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		pids = append(pids, line)
	}

	// Rewind the file to the beginning
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	updatedPIDs := []string{}
	for index, pid := range pids {
		if pid != delete_pid {
			// Create a new slice by excluding the element at the specified index
			_, err = file.WriteString(pid + "\n")
			if err != nil {
				return err
			}
		} else {
			updatedPIDs = append(pids[:index], pids[index+1:]...)
		}
	}

	// Truncate any remaining content (if the new slice is shorter)
	if len(updatedPIDs) < len(pids) {
		err = file.Truncate(int64(len(updatedPIDs)))
		if err != nil {
			return err
		}
	}
	return nil
}

func getCurrentlyUsedChannels(response *APIResponseStruct) (error) {
	// should be nil but its always better to check
	if response.ActiveStreams == nil {
		response.ActiveStreams = &ActiveStreamsStruct{
			Playlists: make(map[string]*PlaylistStruct),
		}
	} else if response.ActiveStreams.Playlists == nil {
		response.ActiveStreams.Playlists = make(map[string]*PlaylistStruct)
	}
	BufferInformation.Range(func(_, value interface{}) bool {
		playlist, ok := value.(Playlist)
		if !ok {
			return true // Skip if the type assertion fails
		}
		
		var playlistID = playlist.PlaylistID
		// should be nil but its always better to check
		if response.ActiveStreams == nil {
			response.ActiveStreams = &ActiveStreamsStruct{
				Playlists: make(map[string]*PlaylistStruct),
			}
		} else if response.ActiveStreams.Playlists == nil {
			response.ActiveStreams.Playlists = make(map[string]*PlaylistStruct)
		}
		response.ActiveStreams.Playlists[playlistID] = createPlaylistStruct(playlist.PlaylistName, playlist.Streams)
		return true
	})
	return nil
}

/*
	This function will extract the info from the ThisStrem Struct
*/
func createPlaylistStruct(name string, streams map[int]ThisStream) *PlaylistStruct{
	var playlist = &PlaylistStruct{}
	playlist.PlaylistName = name
	playlist.ActiveChannels = &[]string{}

	for _, stream := range streams {
		*playlist.ActiveChannels = append(*playlist.ActiveChannels, stream.ChannelName)
	}
	return playlist
}
