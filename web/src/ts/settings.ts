class SettingsCategory {
  DocumentID: string = "content_settings"
  headline: string
  settingsKeys: string

  constructor(headline: string, settingsKeys: string) {
    this.headline = headline
    this.settingsKeys = settingsKeys
  }

  createSettingsCheckbox(settingsKey: string, title: string): any {
    var setting = document.createElement("TR")
    var content: PopupContent = new PopupContent()
    var data = SERVER["settings"][settingsKey]
    var tdLeft = document.createElement("TD")
    tdLeft.innerHTML = title + ":"

    var tdRight = document.createElement("TD")
    var input = content.createCheckbox(settingsKey)
    input.checked = data
    input.setAttribute("onchange", "javascript: this.className = 'changed'")
    tdRight.appendChild(input)

    setting.appendChild(tdLeft)
    setting.appendChild(tdRight)
    return setting
  }

  createCategoryHeadline(value: string): any {
    var element = document.createElement("H4")
    element.innerHTML = value
    return element
  }

  createHR(): any {
    var element = document.createElement("HR")
    return element
  }

  createSettings(settingsKey: string): any {
    var setting = document.createElement("TR")
    var content: PopupContent = new PopupContent()
    var data = SERVER["settings"][settingsKey]

    switch (settingsKey) {

      // Texteingaben
      case "update":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.update.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createInput("text", "update", data.toString())
        input.setAttribute("placeholder", "{{.settings.update.placeholder}}")
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "backup.path":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.backupPath.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createInput("text", "backup.path", data)
        input.setAttribute("placeholder", "{{.settings.backupPath.placeholder}}")
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "temp.path":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.tempPath.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createInput("text", "temp.path", data)
        input.setAttribute("placeholder", "{{.settings.tmpPath.placeholder}}")
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "user.agent":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.userAgent.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createInput("text", "user.agent", data)
        input.setAttribute("placeholder", "{{.settings.userAgent.placeholder}}")
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "buffer.timeout":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.bufferTimeout.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createInput("text", "buffer.timeout", data)
        input.setAttribute("placeholder", "{{.settings.bufferTimeout.placeholder}}")
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "ffmpeg.path":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.ffmpegPath.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createInput("text", "ffmpeg.path", data)
        input.setAttribute("placeholder", "{{.settings.ffmpegPath.placeholder}}")
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "ffmpeg.options":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.ffmpegOptions.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createInput("text", "ffmpeg.options", data)
        input.setAttribute("placeholder", "{{.settings.ffmpegOptions.placeholder}}")
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break
      
        case "vlc.path":
          var tdLeft = document.createElement("TD")
          tdLeft.innerHTML = "{{.settings.vlcPath.title}}" + ":"
  
          var tdRight = document.createElement("TD")
          var input = content.createInput("text", "vlc.path", data)
          input.setAttribute("placeholder", "{{.settings.vlcPath.placeholder}}")
          input.setAttribute("onchange", "javascript: this.className = 'changed'")
          tdRight.appendChild(input)
  
          setting.appendChild(tdLeft)
          setting.appendChild(tdRight)
          break
  
        case "vlc.options":
          var tdLeft = document.createElement("TD")
          tdLeft.innerHTML = "{{.settings.vlcOptions.title}}" + ":"
  
          var tdRight = document.createElement("TD")
          var input = content.createInput("text", "vlc.options", data)
          input.setAttribute("placeholder", "{{.settings.vlcOptions.placeholder}}")
          input.setAttribute("onchange", "javascript: this.className = 'changed'")
          tdRight.appendChild(input)
  
          setting.appendChild(tdLeft)
          setting.appendChild(tdRight)
          break
      
      case "bindingIPs":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.bindingIPs.title}}" + ":"
        var tdRight = document.createElement("TD")
        var input = content.createInput("text", "bindingIPs", data)
        input.setAttribute("id", settingsKey)
        input.addEventListener('click', () => {
          showIPBindingDialogue()
        });
        input.setAttribute("placeholder", "{{.settings.bindingIPs.placeholder}}")
        //input.setAttribute('data-bs-target', '#dialogueContainer')
        //input.setAttribute("data-bs-toggle" , "modal")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "epgCategories":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.epgCategories.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createInput("text", "epgCategories", data.toString())
        input.setAttribute("placeholder", "{{.settings.epgCategories.placeholder}}")
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "epgCategoriesColors":
          var tdLeft = document.createElement("TD")
          tdLeft.innerHTML = "{{.settings.epgCategoriesColors.title}}" + ":"
  
          var tdRight = document.createElement("TD")
          var input = content.createInput("text", "epgCategoriesColors", data.toString())
          input.setAttribute("placeholder", "{{.settings.epgCategoriesColors.placeholder}}")
          input.setAttribute("onchange", "javascript: this.className = 'changed'")
          tdRight.appendChild(input)
  
          setting.appendChild(tdLeft)
          setting.appendChild(tdRight)
          break

      case "threadfinDomain":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.threadfinDomain.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createInput("text", "threadfinDomain", data.toString())
        input.setAttribute("placeholder", "{{.settings.threadfinDomain.placeholder}}")
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)
        
        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "udpxy":
        var tdLeft = document.createElement("TD");
        tdLeft.innerHTML = "{{.settings.udpxy.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createInput("text", "udpxy", data)
        input.setAttribute("placeholder", "{{.settings.udpxy.placeholder}}")
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      // Checkboxen
      case "authentication.web":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.authenticationWEB.title}}")
        break

      case "authentication.pms":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.authenticationPMS.title}}")
        break

      case "authentication.m3u":
        setting = this.createSettingsCheckbox(settingsKey,"{{.settings.authenticationM3U.title}}")
        break

      case "authentication.xml":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.authenticationXML.title}}")
        break

      case "authentication.api":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.authenticationAPI.title}}")
        break

      case "files.update":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.filesUpdate.title}}")
        break

      case "cache.images":
        setting = this.createSettingsCheckbox(settingsKey,"{{.settings.cacheImages.title}}")
        break

      case "xepg.replace.missing.images":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.replaceEmptyImages.title}}")
        break

      case "xepg.replace.channel.title":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.replaceChannelTitle.title}}")
        break

      case "storeBufferInRAM":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.storeBufferInRAM.title}}")
        break

      case "buffer.autoReconnect":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.autoReconnect.title}}")
        break

      case "omitPorts":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.omitPorts.title}}")
        break

      case "forceHttps":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.forceHttps.title}}")
        break

      case "useHttps":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.useHttps.title}}")
        break

      case "forceClientHttps":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.forceClientHttps.title}}")
        break
      
      case "domainUseHttps":
          setting = this.createSettingsCheckbox(settingsKey, "{{.settings.domainUseHttps.title}}")
          break

      case "enableNonAscii":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.enableNonAscii.title}}")
          break

      case "ThreadfinAutoUpdate":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.ThreadfinAutoUpdate.title}}")
        break

      case "ssdp":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.ssdp.title}}")
        break

      case "dummy":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.dummy.title}}")
        break

      case "ignoreFilters":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.ignoreFilters.title}}")
        break

      case "api":
        setting = this.createSettingsCheckbox(settingsKey, "{{.settings.api.title}}")
        break

      // Select
      case "dummyChannel":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.dummyChannel.title}}" + ":"

        var tdRight = document.createElement("TD")
        var text: any[] = ["PPV", "30 Minutes", "60 Minutes", "90 Minutes", "120 Minutes", "180 Minutes", "240 Minutes", "360 Minutes"]
        var values: any[] = ["PPV", "30_Minutes", "60_Minutes", "90_Minutes", "120_Minutes", "180_Minutes", "240_Minutes", "360_Minutes"]

        var select = content.createSelect(text, values, data, settingsKey)
        select.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(select)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "tuner":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.tuner.title}}" + ":"

        var tdRight = document.createElement("TD")
        var text = new Array()
        var values = new Array()

        for (var i = 1; i <= 100; i++) {
          text.push(i)
          values.push(i)
        }

        var select = content.createSelect(text, values, data, settingsKey)
        select.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(select)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "epgSource":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.epgSource.title}}" + ":"

        var tdRight = document.createElement("TD")
        var text: any[] = ["PMS", "XEPG"]
        var values: any[] = ["PMS", "XEPG"]

        var select = content.createSelect(text, values, data, settingsKey)
        select.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(select)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "backup.keep":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.backupKeep.title}}" + ":"

        var tdRight = document.createElement("TD")
        var text: any[] = ["5", "10", "20", "30", "40", "50"]
        var values: any[] = ["5", "10", "20", "30", "40", "50"]

        var select = content.createSelect(text, values, data, settingsKey)
        select.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(select)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "buffer.size.kb":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.bufferSize.title}}" + ":"

        var tdRight = document.createElement("TD")
        var text: any[] = ["0.5 MB", "1 MB", "2 MB", "3 MB", "4 MB", "5 MB", "6 MB", "7 MB", "8 MB"]
        var values: any[] = ["512", "1024", "2048", "3072", "4096", "5120", "6144", "7168", "8192"]

        var select = content.createSelect(text, values, data, settingsKey)
        select.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(select)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "buffer":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.streamBuffering.title}}" + ":"

        var tdRight = document.createElement("TD")
        var text: any[] = ["{{.settings.streamBuffering.info_false}}", "FFmpeg: ({{.settings.streamBuffering.info_ffmpeg}})", "VLC: ({{.settings.streamBuffering.info_vlc}})", "Threadfin: ({{.settings.streamBuffering.info_threadfin}})"]
        var values: any[] = ["-", "ffmpeg", "vlc", "threadfin"]

        var select = content.createSelect(text, values, data, settingsKey)
        select.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(select)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      // Button
      case "uploadCustomImage":

        var tdLeft = document.createElement("TD");
        tdLeft.innerHTML = "{{.settings.uploadCustomImage.title}}" + ":";

        var tdRight = document.createElement("TD");
        var button = content.createInput("button", "upload", "{{.button.uploadCustomImage}}");
        button.setAttribute('onclick', 'javascript: uploadCustomImage();');
        tdRight.appendChild(button)
        setting.appendChild(tdLeft);
        setting.appendChild(tdRight);
        break;
    }

    return setting

  }

  createDescription(settingsKey: string): any {

    var description = document.createElement("TR")
    var text: string
    switch (settingsKey) {

      case "authentication.web":
        text = "{{.settings.authenticationWEB.description}}"
        break

      case "authentication.m3u":
        text = "{{.settings.authenticationM3U.description}}"
        break

      case "authentication.pms":
        text = "{{.settings.authenticationPMS.description}}"
        break

      case "authentication.xml":
        text = "{{.settings.authenticationXML.description}}"
        break

      case "authentication.api":
        if (SERVER["settings"]["authentication.web"] == true) {
          text = "{{.settings.authenticationAPI.description}}"
        }
        break
      
      case "uploadCustomImage":
        text = "{{.settings.uploadCustomImage.description}}";
        break;

      case "ThreadfinAutoUpdate":
        text = "{{.settings.ThreadfinAutoUpdate.description}}"
        break

      case "bindingIPs":
        text = "{{.settings.bindingIPs.description}}"
        break

      case "backup.keep":
        text = "{{.settings.backupKeep.description}}"
        break

      case "backup.path":
        text = "{{.settings.backupPath.description}}"
        break

      case "temp.path":
        text = "{{.settings.tempPath.description}}"
        break

      case "buffer":
        text = "{{.settings.streamBuffering.description}}"
        break

      case "buffer.size.kb":
        text = "{{.settings.bufferSize.description}}"
        break

      case "buffer.autoReconnect":
        text = "{{.settings.autoReconnect.description}}"
        break

      case "storeBufferInRAM":
        text = "{{.settings.storeBufferInRAM.description}}"
        break

      case "omitPorts":
        text = "{{.settings.omitPorts.description}}"
        break

      case "forceHttps":
        text = "{{.settings.forceHttps.description}}"
        break

      case "useHttps":
        text = "{{.settings.useHttps.description}}"
        break

      case "forceClientHttps":
        text = "{{.settings.forceClientHttps.description}}"
        break

      case "threadfinDomain":
          text = "{{.settings.threadfinDomain.description}}"
          break

      case "enableNonAscii":
        text = "{{.settings.enableNonAscii.description}}"
        break

      case "epgCategories":
        text = "{{.settings.epgCategories.description}}"
        break

      case "epgCategoriesColors":
        text = "{{.settings.epgCategoriesColors.description}}"
        break

      case "buffer.timeout":
        text = "{{.settings.bufferTimeout.description}}"
        break

      case "user.agent":
        text = "{{.settings.userAgent.description}}"
        break

      case "ffmpeg.path":
        text = "{{.settings.ffmpegPath.description}}"
        break

      case "ffmpeg.options":
        text = "{{.settings.ffmpegOptions.description}}"
        break

      case "vlc.path":
        text = "{{.settings.vlcPath.description}}"
        break

      case "vlc.options":
        text = "{{.settings.vlcOptions.description}}"
        break

      case "epgSource":
        text = "{{.settings.epgSource.description}}"
        break

      case "tuner":
        text = "{{.settings.tuner.description}}"
        break

      case "update":
        text = "{{.settings.update.description}}"
        break

      case "api":
        text = "{{.settings.api.description}}"
        break

      case "ssdp":
        text = "{{.settings.ssdp.description}}"
        break

      case "files.update":
        text = "{{.settings.filesUpdate.description}}"
        break

      case "cache.images":
        text = "{{.settings.cacheImages.description}}"
        break

      case "xepg.replace.missing.images":
        text = "{{.settings.replaceEmptyImages.description}}"
        break

      case "xepg.replace.channel.title":
        text = "{{.settings.replaceChannelTitle.description}}"
        break

      case "udpxy":
        text = "{{.settings.udpxy.description}}"
        break

      default:
        text = ""
        break

    }

    var tdLeft = document.createElement("TD")
    tdLeft.innerHTML = ""

    var tdRight = document.createElement("TD")
    var pre = document.createElement("PRE")
    pre.innerHTML = text
    tdRight.appendChild(pre)

    description.appendChild(tdLeft)
    description.appendChild(tdRight)

    return description

  }

}

class SettingsCategoryItem extends SettingsCategory {
  
  constructor(headline: string, settingsKeys: string) {
    super(headline, settingsKeys)
  }

  createCategory(): void {
    var doc = document.getElementById(this.DocumentID)
    doc.appendChild(this.createCategoryHeadline(this.headline))

    // Tabelle für die Kategorie erstellen

    var table = document.createElement("TABLE")

    var keys = this.settingsKeys.split(",")

    keys.forEach(settingsKey => {

      switch (settingsKey) {

        case "authentication.pms":
        case "authentication.m3u":
        case "authentication.xml":
        case "authentication.api":
          if (SERVER["settings"]["authentication.web"] == false) {
            break
          }

        default:
          var item = this.createSettings(settingsKey)
          var description = this.createDescription(settingsKey)

          table.appendChild(item)
          table.appendChild(description)
          break

      }

    });

    doc.appendChild(table)
    doc.appendChild(this.createHR())
  }

}

function showSettings() {
  console.log("SETTINGS");

  for (let i = 0; i < settingsCategory.length; i++) {
    settingsCategory[i].createCategory()
  }

}

function saveSettings() {
  console.log("Save Settings");

  var cmd = "saveSettings"
  var div = document.getElementById("content_settings")
  var settings = div.getElementsByClassName("changed")

  var newSettings = new Object();

  for (let i = 0; i < settings.length; i++) {

    var name: string
    var value: any

    switch (settings[i].tagName) {
      case "INPUT":

        switch ((settings[i] as HTMLInputElement).type) {
          case "checkbox":
            name = (settings[i] as HTMLInputElement).name
            value = (settings[i] as HTMLInputElement).checked
            newSettings[name] = value
            break

          case "text":
            name = (settings[i] as HTMLInputElement).name
            value = (settings[i] as HTMLInputElement).value

            switch (name) {
              case "update":
                value = value.split(",")
                value = value.filter(function (e: any) { return e })
                break

              case "buffer.timeout":
                value = parseFloat(value)
            }

            newSettings[name] = value
            break
          }

        break

      case "SELECT":
        name = (settings[i] as HTMLSelectElement).name
        value = (settings[i] as HTMLSelectElement).value

        // Wenn der Wert eine Zahl ist, wird dieser als Zahl gespeichert
        if (isNaN(value)) {
          newSettings[name] = value
        } else {
          newSettings[name] = parseInt(value)
        }

        break

    }

  }

  var data = new Object()
  data["settings"] = newSettings

  var server: Server = new Server(cmd)
  server.request(data)
}

function uploadCustomImage() {
  if (document.getElementById('upload')) {
    document.getElementById('upload').remove()
  }

  var upload = document.createElement("INPUT");
  upload.setAttribute("type", "file");
  upload.setAttribute("accept", ".jpg,.png")
  upload.setAttribute("class", "notVisible");
  upload.setAttribute("name", "");
  upload.id = "upload";

  document.body.appendChild(upload);
  upload.click();

  upload.onblur = function () {
    alert()
  }

  upload.onchange = function () {

    var filename = (upload as HTMLInputElement).files[0].name

    var reader = new FileReader();
    var file = (document.querySelector('input[type=file]') as HTMLInputElement).files[0];

    if (file) {

      reader.readAsDataURL(file);
      reader.onload = function () {
        console.log(reader.result);
        var data = new Object();
        var cmd = "uploadCustomImage"
        data["base64"] = reader.result
        data["filename"] = file.name

        var server: Server = new Server(cmd)
        server.request(data)

        var updateLogo = (document.getElementById('update-icon') as HTMLInputElement)
        updateLogo.checked = false
        updateLogo.className = "changed"

      };

    } else {
      alert("File could not be loaded")
    }

    upload.remove()
    return
  }

}