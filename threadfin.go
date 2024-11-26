// Copyright 2019 marmei. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the
// LICENSE file.
// GitHub: https://github.com/Threadfin/Threadfin

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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
var GitHub = GitHubStruct{Branch: "Main", User: "marcelGoerentz", Repo: "Threadfin", Update: true}

/*
	Branch: GitHub Branch
	User: 	GitHub Username
	Repo: 	GitHub Repository
	Update: Automatic updates from the GitHub repository [true|false]
*/

// Name : Programmname
const Name = "Threadfin"

// Version : Version, die Build Nummer wird in der main func geparst.
const Version = "1.7.11-beta"

// DBVersion : Datanbank Version
const DBVersion = "0.5.0"

// APIVersion : API Version
const APIVersion = "2.0.0-beta"

var homeDirectory = fmt.Sprintf("%s%s.%s%s", src.GetUserHomeDirectory(), string(os.PathSeparator), strings.ToLower(Name), string(os.PathSeparator))
var samplePath = fmt.Sprintf("%spath%sto%sthreadfin%s", string(os.PathSeparator), string(os.PathSeparator), string(os.PathSeparator), string(os.PathSeparator))
var sampleRestore = fmt.Sprintf("%spath%sto%sfile%s", string(os.PathSeparator), string(os.PathSeparator), string(os.PathSeparator), string(os.PathSeparator))

var configFolder = flag.String("config", "", ": Config Folder        ["+samplePath+"] (default: "+homeDirectory+")")
var port = flag.Int("port", 34400, ": Server port")
var useHttps = flag.Bool("useHttps", false , ": Use Https Webserver [place server.crt and server.key in config folder]")
var restore = flag.String("restore", "", ": Restore from backup  ["+sampleRestore+"threadfin_backup.zip]")

var gitBranch = flag.String("branch", "", ": Git Branch           [main|beta] (default: main)")
var debug = flag.Int("debug", 0, ": Debug level          [0 - 3] (default: 0)")
var info = flag.Bool("info", false, ": Show system info")
var h = flag.Bool("h", false, ": Show help")

// Aktiviert den Entwicklungsmodus. Für den Webserver werden dann die lokalen Dateien verwendet.
var dev = flag.Bool("dev", false, ": Activates the developer mode, the source code must be available. The local files for the web interface are used.")

func main() {

	// Build-Nummer von der Versionsnummer trennen
	var build = strings.Split(Version, ".")

	var system = &src.System
	system.APIVersion = APIVersion
	system.Branch = strings.ToTitle(GitHub.Branch)
	system.Build = build[len(build)-1:][0]
	system.DBVersion = DBVersion
	system.GitHub = GitHub
	system.Name = Name
	system.Version = strings.Join(build[0:len(build)-1], ".")

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

	// Systeminformationen anzeigen
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

	// Webserver Port
	if *port != 0 {
		system.Flag.Port = fmt.Sprintf("%d", *port)
	}

	// Https Webserver
	system.Flag.UseHttps = *useHttps


	// Kill all remaining processes and remove PIDs file
	tempFolder := os.TempDir() + string(os.PathSeparator) +  strings.ToLower(Name) + string(os.PathSeparator)
	folders, err := os.ReadDir(tempFolder)
	if err == nil {
		for _, folder := range folders {
			folderName := fmt.Sprintf("%s%s", tempFolder, folder.Name())
			pids, err := getPIDsFromFile(folderName)
			if err != nil {
				fmt.Printf("Error scanning file PIDs: %v", err)
			} else {
				if len(pids) > 0 {
					for _, pid := range pids {
						err := killProcess(pid)
						if err != nil {
							fmt.Printf("Error killing process %s: %v", pid, err)
						} else {
							fmt.Printf("Successfully killed process %s", pid)
						}
					}
					os.Remove(folderName + string(os.PathSeparator) + "PIDs")
				}
			}
		}
	}

	// Branch
	system.Flag.Branch = *gitBranch
	if len(system.Flag.Branch) > 0 {
		fmt.Println("Git Branch is now:", system.Flag.Branch)
	}

	// Debug Level
	system.Flag.Debug = *debug
	if system.Flag.Debug > 3 {
		flag.Usage()
		return
	}

	// Speicherort für die Konfigurationsdateien
	if len(*configFolder) > 0 {
		system.Folder.Config = *configFolder
	}

	// Backup wiederherstellen
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

	err = src.Init()
	if err != nil {
		src.ShowError(err, 0)
		os.Exit(0)
	}

	err = src.BinaryUpdate()
	if err != nil {
		src.ShowError(err, 0)
	}

	err = src.StartSystem(false)
	if err != nil {
		src.ShowError(err, 0)
		os.Exit(0)
	}

	err = src.InitMaintenance()
	if err != nil {
		src.ShowError(err, 0)
		os.Exit(0)
	}

	err = src.StartWebserver()
	if err != nil {
		src.ShowError(err, 0)
		os.Exit(0)
	}

}

func getPIDsFromFile(tempFolder string) ([]string, error){
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
