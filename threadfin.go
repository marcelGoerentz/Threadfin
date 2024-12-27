// Copyright 2019 marmei. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the
// LICENSE file.
// GitHub: https://github.com/Threadfin/Threadfin

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"threadfin/src"
)

// GitHubStruct : GitHub Account. Über diesen Account werden die Updates veröffentlicht
type GitHubStruct struct {
	Branch  string
	Repo    string
	Update  bool
	User    string
	TagName string
}

// GitHub : GitHub Account
// If you want to fork this project, enter your Github account here. This prevents a newer version of Threadfin from updating your version.
var GitHub = GitHubStruct{Branch: "master", User: "marcelGoerentz", Repo: "Threadfin", Update: true}

/*
	Branch: GitHub Branch
	User: 	GitHub Username
	Repo: 	GitHub Repository
	Update: Automatic updates from the GitHub repository [true|false]
*/

// Name : Programmname
const Name = "Threadfin"

// Version : Version, die Build Nummer wird in der main func geparst.
const Version = "1.8.0.0"

// DBVersion : Datanbank Version
const DBVersion = "0.5.0"

// APIVersion : API Version
const APIVersion = "2.0.0"

var homeDirectory = fmt.Sprintf("%s%s.%s%s", src.GetUserHomeDirectory(), string(os.PathSeparator), strings.ToLower(Name), string(os.PathSeparator))
var samplePath = fmt.Sprintf("%spath%sto%sthreadfin%s", string(os.PathSeparator), string(os.PathSeparator), string(os.PathSeparator), string(os.PathSeparator))
var sampleRestore = fmt.Sprintf("%spath%sto%sfile%s", string(os.PathSeparator), string(os.PathSeparator), string(os.PathSeparator), string(os.PathSeparator))

var configFolder = flag.String("config", "", ": Config Folder        ["+samplePath+"] (default: "+homeDirectory+")")
var port = flag.Int("port", 34400, ": Server port")
var useHttps = flag.Bool("useHttps", false, ": Use Https Webserver [place server.crt and server.key in config folder]")
var restore = flag.String("restore", "", ": Restore from backup  ["+sampleRestore+"threadfin_backup.zip]")

var betaFlag = flag.Bool("beta", false, ": Beta flag           [true|false] (default: false)")
var debug = flag.Int("debug", 0, ": Debug level          [0 - 3] (default: 0)")
var info = flag.Bool("info", false, ": Show system info")
var h = flag.Bool("h", false, ": Show help")

// Aktiviert den Entwicklungsmodus. Für den Webserver werden dann die lokalen Dateien verwendet.
var dev = flag.Bool("dev", false, ": Activates the developer mode, the source code must be available. The local files for the web interface are used.")

func main() {

	webserver := &src.WebServer{}

	// Handle signals
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGABRT, syscall.SIGTERM, syscall.SIGHUP)
	go handleSignals(sigs, done, webserver)

	// Split build from version string
	var build = strings.Split(Version, ".")

	var system = &src.System
	system.APIVersion = APIVersion
	system.Branch = strings.ToTitle(GitHub.Branch)
	system.Build = build[len(build)-1:][0]
	system.DBVersion = DBVersion
	system.GitHub = GitHub
	system.Name = Name
	system.Version = strings.Join(build[0:len(build)-1], ".")

	// Check which version has been build
	if Beta {
		system.Beta = true
	}

	// Panic !!!
	defer func() {

		if r := recover(); r != nil {

			fmt.Println()
			fmt.Println("* * * * * FATAL ERROR * * * * *")
			fmt.Println("OS:  ", runtime.GOOS)
			fmt.Println("Arch:", runtime.GOARCH)
			fmt.Println("Err: ", r)
			fmt.Println()

			pc := make([]uintptr, 20)
			runtime.Callers(2, pc)

			for i := range pc {

				if runtime.FuncForPC(pc[i]) != nil {

					f := runtime.FuncForPC(pc[i])
					file, line := f.FileLine(pc[i])

					if string(file)[0:1] != "?" {
						fmt.Printf("%s:%d %s\n", filepath.Base(file), line, f.Name())
					}

				}

			}

			fmt.Println()
			fmt.Println("* * * * * * * * * * * * * * * *")

		}

	}()

	flag.Parse()

	if *h {
		flag.Usage()
		return
	}

	system.Dev = *dev

	// Show system information
	if *info {

		system.Flag.Info = true

		err := src.Init()
		if err != nil {
			src.ShowError(err, 0)
			os.Exit(0)
		}

		src.ShowSystemInfo()
		return

	}

	// Webserver port
	if *port != 0 {
		system.Flag.Port = fmt.Sprintf("%d", *port)
	}

	// Https webserver
	system.Flag.UseHttps = *useHttps

	// Kill all remaining processes and remove PIDs file
	killAllProcesses()

	// Debug Level
	system.Flag.Debug = *debug
	if system.Flag.Debug > 3 {
		flag.Usage()
		return
	}

	// Config folder place
	if len(*configFolder) > 0 {
		system.Folder.Config = *configFolder
	}

	// Restore from backup
	if len(*restore) > 0 {

		system.Flag.Restore = *restore

		err := src.Init()
		if err != nil {
			src.ShowError(err, 0)
			os.Exit(0)
		}

		err = src.ThreadfinRestoreFromCLI(*restore)
		if err != nil {
			src.ShowError(err, 0)
		}
		os.Exit(0)
	}

	// Initialize threadfin
	err := src.Init()
	if err != nil {
		src.ShowError(err, 0)
		os.Exit(0)
	}

	// Update binary
	err = src.BinaryUpdate()
	if err != nil {
		src.ShowError(err, 0)
	}

	// Build the database
	err = src.StartSystem(false)
	if err != nil {
		src.ShowError(err, 0)
		os.Exit(0)
	}

	// Update playlists and xml files
	err = src.InitMaintenance()
	if err != nil {
		src.ShowError(err, 0)
		os.Exit(0)
	}

	// Start the Webserver
	srv, err := src.StartWebserver()
	if err != nil {
		src.ShowError(err, 0)
		os.Exit(0)
	}

	// Add references to the structure so everything can be exited gracefully
	webserver.Server = srv.Server
	webserver.SM = srv.SM

	// Wait for the Signal to end the program
	<-done
	src.ShowInfo("Threadfin:Exiting Threadfin")
}

/*
handleSignals should be called in a go routine and will handle incoming system signals
It will make sure that all running processes will be killed before exiting the program
*/
func handleSignals(sigs chan os.Signal, done chan bool, webserver *src.WebServer) {
	for sig := range sigs {
		switch sig {
		case syscall.SIGHUP:
			src.ShowInfo("Threadfin:Updating configuration")
			continue
		case syscall.SIGINT, syscall.SIGABRT, syscall.SIGTERM:

			src.ShowInfo("Threadfin:Stop all streams")
			// Stop all streams
			stopAllStreams(webserver)

			src.ShowInfo("Threadfin:Killing all processes")
			// Kill all processes
			killAllProcesses()

			// Shutdown the webserver gracefully
			shutdownWebserver(webserver)

			// Send signal that everything has ended
			done <- true
			return
		default:
			src.ShowDebug("Threadfin: Uncatched signal!", 1)
			continue
		}
	}
}

func stopAllStreams(webserver *src.WebServer) {
	if webserver != nil {
		if webserver.SM != nil {
			if webserver.SM.Playlists != nil {
				for playlistID, playlist := range webserver.SM.Playlists {
					for streamID, stream := range playlist.Streams {
						for clientID := range stream.Clients {
							webserver.SM.StopStream(playlistID, streamID, clientID)
						}
					}
				}
			}
		}
	}
}

/*
getTempFolder will get the first temp folder within the threadfin temp folder or returns an empty string if there is no folder
*/
func getTempFolder() string {
	tempFolder := os.TempDir() + string(os.PathSeparator) + strings.ToLower(Name) + string(os.PathSeparator)
	folders, err := os.ReadDir(tempFolder)
	if err == nil {
		for _, folder := range folders {
			return fmt.Sprintf("%s%s", tempFolder, folder.Name())
		}
	}
	return ""
}

/*
getPIDsFromFile will open the PIDs file within the given Folder and returns the list of PIDs saved in it
*/
func getPIDsFromFile(tempFolder string) ([]string, error) {
	pids := []string{}
	// Open the file
	pidsFile := tempFolder + string(os.PathSeparator) + "PIDs"
	_, err_stat := os.Stat(pidsFile)
	if os.IsNotExist(err_stat) {
		return pids, nil // Return early if the file doesn't exist
	}

	file, err_open := os.Open(pidsFile)
	if err_open != nil {
		return nil, err_open
	}
	defer file.Close() // Close the file when done

	// Create a scanner
	scanner := bufio.NewScanner(file)

	// Read line by line
	for scanner.Scan() {
		line := scanner.Text()
		pids = append(pids, line)
	}

	return pids, nil
}

// killProcess kills a process by its PID
func killProcess(pid string) error {
	cmd := exec.Command("kill", "-9", pid)
	return cmd.Run()
}

//
func killAllProcesses() {
	tempFolder := getTempFolder()
	if tempFolder != "" {
		pids, err := getPIDsFromFile(tempFolder)
		if err == nil {
			for _, pid := range pids {
				src.ShowDebug(fmt.Sprintf("Threadfin:Killing process: %s\n", pid), 1)
				killProcess(pid)
			}
		}
	}
}

func shutdownWebserver(webserver *src.WebServer) {
	if webserver.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		err := webserver.Server.Shutdown(ctx)
		if err != nil {
			src.ShowError(err, 0) // TODO: Add error code
		}
	}
}
