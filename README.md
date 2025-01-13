<div align="center" style="background-color: #111; padding: 100;">
    <a href="https://github.com/marcelGoerentz/Threadfin"><img width="285" height="80" src="web/public/img/threadfin.png" alt="Threadfin" /></a>
</div>
<br>

# Threadfin
## Discord
https://discord.gg/gwkpMxepPA

## M3U Proxy for Plex DVR and Emby/Jellyfin Live TV. Based on xTeVe.

You can follow the old xTeVe documentation for now until I update it for Threadfin. Documentation for setup and configuration is [here](https://github.com/xteve-project/xTeVe-Documentation/blob/master/en/configuration.md).

A wiki is already in creation!!!
Thanks to @Renkyz

#### Donation
Buy me a coffee with
[Paypal/Me](https://paypal.me/MarcelGoerentz)

## Requirements
### Plex
* Plex Media Server (1.11.1.4730 or newer)
* Plex Client with DVR support
* Plex Pass

### Emby
* Emby Server (3.5.3.0 or newer)
* Emby Client with Live-TV support
* Emby Premiere

### Jellyfin
* Jellyfin Server (10.7.1 or newer)
* Jellyfin Client with Live-TV support

--- 

## Threadfin Features

### Backend (Server)

### Webserver
* Listening IPs and port is configurable
* HTTP or HTTPS server can be used
* Fits to run behind a reverse proxy

#### Files
* Merge external M3U files
* Merge external XMLTV files
* Automatic M3U and XMLTV update
* M3U and XMLTV export

#### Channel management
* Filtering streams
* Channel mapping
* Channel order
* Channel logos
* Channel categories

#### Streaming
* RAM buffer or file based buffer configurable
* Buffer with HLS / M3U8 support
* Proxies a stream to multiple clients
* Number of tuners adjustable
* Compatible with Plex / Emby / Jellyfin EPG
* Customizable image will be displayed when the tuner limit has been reached

### Frontend (Webclient)

### UI
* New Bootstrap based UI
* "Back to Top" button

#### Filter Group
* Can now add a starting channel number for the filter group

#### Map Editor
* Can now multi select Bulk Edit by holding shift
* Now has a separate table for inactive channels
* Can add 3 backup channels for an active channel (backup channels do NOT have to be active)
* Alpha Numeric sorting now sorts correctly
* Can now add a starting channel number for Bulk Edit to renumber multiple channels at a time
* PPV channels can now map the channel name to an EPG

---

## CLI arguments

These are the currently available command line arguments:

| arg        | type    | description                                             | example                                     |
|:-----------|:--------|:--------------------------------------------------------|:--------------------------------------------|
| -h         | bool    | prints the help and don't start the service             | -h                                          |
| -dev       | bool    | activates the developer mode                            | -dev                                        |
| -config    | string  | sets the path to the root config folder                 | -config=~./.threadfin                       |
| -port      | integer | sets the port for the webserver (also for https)        | -port=34400                                 |
| -useHttps  | bool    | switches the webserver to https                         | -useHttps                                   |
| -restore   | string  | restores the settings from the given filepath           | -restore=/path/to/file/threadfin_backup.zip |
| -debug     | integer | sets the debug level                                    | -debug=3                                    |
| -info      | bool    | prints the system info                                  | -info                                       |

---

## Docker Image
[Threadfin on Docker Hub](https://hub.docker.com/r/mgoerentz/threadfin)

* Docker compose example

```yaml
version: "2.3"
services:
  threadfin:
    image: mgoerentz/threadfin:latest
    container_name: threadfin
    ports:
      - 34400:34400
    environment:
      - PUID=1001
      - PGID=1001
      - TZ=America/Los_Angeles
    volumes:
      - ./data/conf:/home/threadfin/conf
      - ./data/temp:/tmp/threadfin:rw
    restart: unless-stopped
networks:{}
```

* Docker compose example with gluetun (VPN)

```yaml
version: "3.8"
services:
  gluetun:
    container_name: gluetun
    image: qmcgaw/gluetun
    devices:
      - /dev/net/tun:/dev/net/tun
    cap_add:
      - NET_ADMIN
    ports:
      - 8000:8000
      - 34400:34400
    volumes:
      - ./gluetun/data:/gluetun
    environment:
      - VPN_SERVICE_PROVIDER=private internet access # Visit https://github.com/qdm12/gluetun-wiki to apply settings from your provider
      - OPENVPN_USER=<username>
      - OPENVPN_PASSWORD=<password>
      - SERVER_REGIONS=Netherlands
      - VPN_PORT_FORWARDING=on
      - VPN_PORT_FORWARDING_PROVIDE=private internet access
    restart: unless-stopped
  threadfin:
    image: mgoerentz/threadfin:latest
    container_name: threadfin
    environment:
      - PUID=1000 # Add your HOST User ID
      - PGID=1000 # Add your HOST User GROUP
      - THREADFIN_PORT=34400
      - TZ=Europe/Berlin
    volumes:
      - ./data/conf:/home/threadfin/conf
      - ./data/tmp:/tmp/threadfin:rw
    restart: unless-stopped
    depends_on:
      gluetun:
        condition: service_healthy
        restart: true
    network_mode: service:gluetun
networks: {}
```


---

### Threadfin Beta branch
New features and bug fixes are only available in beta branch. Only after successful testing they are merged into the main branch.

**It is not recommended to use the beta version in a production system.**  

#### Switch from release to beta version:
You can switch to the latest beta version by opening the web client of threadfin.
Clicking on "Server Information" and then on "Change to beta version" button on the bottom of the dialogue
Threadfin will then download the latest binary suitable for your OS from Github and restarts the application.

#### Switch from beta to release version:
This is also working like switching from master to beta version. When you are using a beta version the button will show "Change to release version".

When the version is changed, an update is only performed if there is a new version and the update function is activated in the settings.  

---

## Build from source code [Go / Golang]

#### Requirements
* [Go](https://golang.org) (go 1.23 or newer)

#### Dependencies
* [avfs: avfs](https://github.com/avfs/avfs)
* [google: uuid](github.com/google/uuid)
*	[gorilla: websocket](github.com/gorilla/websocket)
*	[hashicorp: go-version](github.com/hashicorp/go-version)
*	[koron: go-ssdp](github.com/koron/go-ssdp)
*	[x: net](golang.org/x/net)
*	[x: text](golang.org/x/text)


#### Build
1. Download source code
2. Install dependencies
```sh
go mod tidy && go mod vendor
```
3. Build Threadfin
```sh
go build . # => relase version
go build -tags beta # => beta version
```

4. Update web files (optional)
If TypeScript files were changed, run:

```sh
tsc -p ./web/tsconfig.json
```

5. Then, to embed updated JavaScript files into the source code (src/webUI.go), run it in development mode at least once:

```sh
go build .
threadfin -dev
```
```pwsh
go build .
threadfin.exe -dev
```

---

## How can I contribute
You can translate the /web/public/lang/en.json file into your mother tongue.

Or you can fork this repo and create a PR for your changes.

---

## Fork without pull request :mega:
When creating a fork, the Threadfin GitHub account must be changed from the source code or the update function disabled.
Future updates of Threadfin would update the Threadfin binary from the original project.

threadfin.go - Line: 36
```Go
var GitHub = GitHubStruct{Branch: "main", User: "Threadfin", Repo: "Threadfin", Update: true}

/*
  Branch: GitHub Branch
  User:   GitHub Username
  Repo:   GitHub Repository
  Update: Automatic updates from the GitHub repository [true|false]
*/

```

This repo also comes with two workflows:
* **The release workflow:** Triggered when something has been pushed to master branch
* **The beta version workflow:** Triggered when something has been pushed to beta branch

There are also 3 utility scripts:
* **create_binaries.sh:** This will automatically generates the binaries for the different platforms (Windows,Linux,FreeBSD,..)
* **set_build_number.sh:** This script will update the buildnumber befor generating the binary
* **update_build_number_variable.sh:** This script will push the updated build number to GitHub

