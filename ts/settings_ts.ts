class SettingsCategory {
  DocumentID:string = "content_settings"
  createCategoryHeadline(value:string):any {
    var element = document.createElement("H4")
    element.innerHTML = value
    return element
  }

  createHR():any {
    var element = document.createElement("HR")
    return element
  }

  createSettings(settingsKey:string):any {
    var setting = document.createElement("TR")
    var content:PopupContent = new PopupContent()
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

        // Checkboxen
      case "authentication.web":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.authenticationWEB.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createCheckbox(settingsKey)
        input.checked = data
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "authentication.pms":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.authenticationPMS.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createCheckbox(settingsKey)
        input.checked = data
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "authentication.m3u":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.authenticationM3U.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createCheckbox(settingsKey)
        input.checked = data
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "authentication.xml":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.authenticationXML.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createCheckbox(settingsKey)
        input.checked = data
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "authentication.api":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.authenticationAPI.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createCheckbox(settingsKey)
        input.checked = data
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "files.update":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.filesUpdate.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createCheckbox(settingsKey)
        input.checked = data
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "xteveAutoUpdate":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.xteveAutoUpdate.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createCheckbox(settingsKey)
        input.checked = data
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        var checkButton = content.createInput("button", "", "{{.button.checkForUpdates}}")
        checkButton.setAttribute("onclick", "javascript: checkForUpdates();")
        tdRight.appendChild(checkButton)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "cache.images":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.cacheImages.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createCheckbox(settingsKey)
        input.checked = data
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "xepg.replace.missing.images":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.replaceEmptyImages.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createCheckbox(settingsKey)
        input.checked = data
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "api":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.api.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createCheckbox(settingsKey)
        input.checked = data
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

        // Select
      case "tuner":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.tuner.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createInput("number", settingsKey, data)
        input.setAttribute("min", "1")
        input.setAttribute("max", "100")
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "source.check.interval":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.sourceCheckInterval.title}}" + ":"

        var tdRight = document.createElement("TD")
        var input = content.createInput("number", settingsKey, data)
        input.setAttribute("min", "0")
        input.setAttribute("max", "1440")
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(input)

        setting.appendChild(tdLeft)
        setting.appendChild(tdRight)
        break

      case "epgSource":
        var tdLeft = document.createElement("TD")
        tdLeft.innerHTML = "{{.settings.epgSource.title}}" + ":"

        var tdRight = document.createElement("TD")
        var text:any[] = ["PMS", "XEPG"]
        var values:any[] = ["PMS", "XEPG"]

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
        var text:any[] = ["5", "10", "20", "30", "40", "50"]
        var values:any[] = ["5", "10", "20", "30", "40", "50"]

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
        var text:any[] = ["0.5 MB", "1 MB", "2 MB", "3 MB", "4 MB", "5 MB", "6 MB", "7 MB", "8 MB"]
        var values:any[] = ["512", "1024", "2048", "3072", "4096", "5120", "6144", "7168", "8192"]

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
        var text:any[] = ["{{.settings.streamBuffering.info_false}}", "xTeVe Reborn: ({{.settings.streamBuffering.info_xteve}})", "FFmpeg: ({{.settings.streamBuffering.info_ffmpeg}})", "VLC: ({{.settings.streamBuffering.info_vlc}})"]
        var values:any[] = ["-", "xteve", "ffmpeg", "vlc"]

        var select = content.createSelect(text, values, data, settingsKey)
        select.setAttribute("onchange", "javascript: this.className = 'changed'")
        tdRight.appendChild(select)

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

    }

    return setting

  }


  createDescription(settingsKey:string):any {

    var description = document.createElement("TR")
    var text:string
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

      case "source.check.interval":
        text = "{{.settings.sourceCheckInterval.description}}"
        break

      case "update":
        text = "{{.settings.update.description}}"
        break

      case "api":
        text = "{{.settings.api.description}}"
        break

      case "files.update":
        text = "{{.settings.filesUpdate.description}}"
        break

      case "xteveAutoUpdate":
        text = "{{.settings.xteveAutoUpdate.description}}"
        break

      case "cache.images":
        text = "{{.settings.cacheImages.description}}"
        break

      case "xepg.replace.missing.images":
        text = "{{.settings.replaceEmptyImages.description}}"
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
  headline:string
  settingsKeys:string

  constructor(headline:string, settingsKeys:string) {
    super()
    this.headline = headline
    this.settingsKeys = settingsKeys
  }

  createCategory():void {
    var headline = this.createCategoryHeadline(this.headline)
    var settingsKeys = this.settingsKeys

    var doc = document.getElementById(this.DocumentID)
    doc.appendChild(headline)

    // Tabelle für die Kategorie erstellen

    var table = document.createElement("TABLE")

    var keys = settingsKeys.split(",")

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

  clearSettingsDirty()

  for (let i = 0; i < settingsCategory.length; i++) {
    settingsCategory[i].createCategory()
  }

  var div = document.getElementById("content_settings")
  div.addEventListener("input", markSettingsDirty)
  div.addEventListener("change", markSettingsDirty)

}

// Unsaved-changes tracking for the Settings page: a bar at the bottom of
// the screen appears as soon as anything is edited, and leaving the page
// (another menu item, or closing the tab) asks before throwing edits away.
var SETTINGS_DIRTY:boolean = false

function markSettingsDirty():void {

  SETTINGS_DIRTY = true

  var bar = document.getElementById("unsavedBar")

  if (!bar) {
    bar = document.createElement("DIV")
    bar.id = "unsavedBar"

    var message = document.createElement("SPAN")
    message.textContent = "{{.settings.unsaved.message}}"
    bar.appendChild(message)

    var discard = document.createElement("INPUT")
    discard.setAttribute("type", "button")
    discard.setAttribute("value", "{{.settings.unsaved.discard}}")
    discard.setAttribute("class", "cancel")
    discard.onclick = discardSettings
    bar.appendChild(discard)

    var save = document.createElement("INPUT")
    save.setAttribute("type", "button")
    save.setAttribute("value", "{{.button.save}}")
    save.setAttribute("class", "save")
    save.onclick = saveSettings
    bar.appendChild(save)

    document.body.appendChild(bar)
  }

  showElement("unsavedBar", true)
}

function clearSettingsDirty():void {

  SETTINGS_DIRTY = false

  if (document.getElementById("unsavedBar")) {
    showElement("unsavedBar", false)
  }
}

function discardSettings():void {

  clearSettingsDirty()

  for (var i = 0; i < menuItems.length; i++) {
    if (menuItems[i]["menuKey"] == "settings") {
      document.getElementById(menuItems[i].id).click()
      return
    }
  }
}

window.addEventListener("beforeunload", function(e) {
  if (SETTINGS_DIRTY) {
    e.preventDefault()
    e.returnValue = ""
  }
})

function saveSettings() {
  console.log("Save Settings");

  // Cleared up front: the server's reply re-opens the Settings menu, which
  // would otherwise trigger the "discard unsaved changes?" prompt.
  clearSettingsDirty()

  var cmd = "saveSettings"
  var div = document.getElementById("content_settings")
  var settings = div.getElementsByClassName("changed")

  var newSettings = new Object();

  for (let i = 0; i < settings.length; i++) {

    var name:string
    var value:any

    switch (settings[i].tagName) {
      case "INPUT":

        switch ((settings[i] as HTMLInputElement).type) {
          case "checkbox":
            name = (settings[i] as HTMLInputElement).name
            value = (settings[i] as HTMLInputElement).checked
            newSettings[name] = value
            break

          case "number":
            name = (settings[i] as HTMLInputElement).name
            value = parseInt((settings[i] as HTMLInputElement).value)

            if (isNaN(value)) {
              showToast(name + ": {{.alert.missingInput}}", "error")
              return
            }

            newSettings[name] = value
            break

          case "text":
            name = (settings[i] as HTMLInputElement).name
            value = (settings[i] as HTMLInputElement).value

            switch (name) {
              case "update":
                value = value.split(",")
                value = value.filter(function(e:any) { return e})
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
        if(isNaN(value)){
          newSettings[name] = value
        } else {
          newSettings[name] = parseInt(value)
        }

        break

    }

  }

  var data = new Object()
  data["settings"] = newSettings

  var server:Server = new Server(cmd)
  server.request(data)
}
