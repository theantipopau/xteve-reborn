class WizardCategory {
  DocumentID = "content"

  createCategoryHeadline(value:string):any {
    var element = document.createElement("H4")
    element.innerHTML = value
    return element
  }
}

class WizardItem extends WizardCategory {
  key:string
  headline:string

  constructor(key:string, headline:string) {
    super()
    this.headline = headline
    this.key = key
  }

  createWizard():void {
    var headline = this.createCategoryHeadline(this.headline)
    var key = this.key
    var content:PopupContent = new PopupContent()
    var description:string

    var doc = document.getElementById(this.DocumentID)
    doc.innerHTML = ""
    doc.appendChild(headline)

    switch (key) {
      case "tuner":
        var input = content.createInput("number", key, "1")
        input.setAttribute("min", "1")
        input.setAttribute("max", "100")
        input.setAttribute("class", "wizard")
        input.id = key
        doc.appendChild(input)

        description = "{{.wizard.tuner.description}}"

        break;

      case "webAuth":
        var text:any[] = ["{{.wizard.webAuth.yes}}", "{{.wizard.webAuth.no}}"]
        var values:any[] = ["true", "false"]

        var select = content.createSelect(text, values, "true", key)
        select.setAttribute("class", "wizard")
        select.id = key
        doc.appendChild(select)

        description = "{{.wizard.webAuth.description}}"

        break
      
      case "epgSource":
        var text:any[] = ["PMS", "XEPG"]
        var values:any[] = ["PMS", "XEPG"]

        var select = content.createSelect(text, values, "XEPG", key)
        select.setAttribute("class", "wizard")
        select.id = key
        doc.appendChild(select)

        description = "{{.wizard.epgSource.description}}"

        break

      case "m3u":
        var input = content.createInput("text", key, "")
        input.setAttribute("placeholder", "{{.wizard.m3u.placeholder}}")
        input.setAttribute("class", "wizard")
        input.id = key
        doc.appendChild(input)

        description = "{{.wizard.m3u.description}}"

        break

      case "xmltv":
        var input = content.createInput("text", key, "")
        input.setAttribute("placeholder", "{{.wizard.xmltv.placeholder}}")
        input.setAttribute("class", "wizard")
        input.id = key
        doc.appendChild(input)

        description = "{{.wizard.xmltv.description}}"

      break

      case "finish":
        var address = SERVER["clientInfo"]["DVR"]

        var box = document.createElement("DIV")
        box.className = "wizard-address-box"

        var code = document.createElement("CODE")
        code.innerText = address
        box.appendChild(code)

        var copyButton = document.createElement("INPUT")
        copyButton.setAttribute("type", "button")
        copyButton.setAttribute("value", "{{.wizard.finish.copyButton}}")
        copyButton.onclick = function() {
          copyTextToClipboard(address)
        }
        box.appendChild(copyButton)

        doc.appendChild(box)

        var steps = document.createElement("UL")
        steps.className = "wizard-steps"

        var plexStep = document.createElement("LI")
        plexStep.innerHTML = "{{.wizard.finish.plexSteps}}"
        steps.appendChild(plexStep)

        var embyStep = document.createElement("LI")
        embyStep.innerHTML = "{{.wizard.finish.embySteps}}"
        steps.appendChild(embyStep)

        doc.appendChild(steps)

        description = "{{.wizard.finish.description}}"

        var nextButton = document.getElementById("next") as HTMLInputElement
        nextButton.value = "{{.wizard.finish.button}}"
        nextButton.onclick = finishWizard

      break

      default:
        console.log(key)
        break;
    }

    var pre = document.createElement("PRE")
    pre.innerHTML = description
    doc.appendChild(pre)

    console.log(headline, key)
  }


}


function readyForConfiguration(wizard:number) {

  var server:Server = new Server("getServerConfig")
  server.request(new Object())

  showElement("loading", false)

  configurationWizard[wizard].createWizard()

}

function saveWizard() {

  var cmd = "saveWizard"
  var div = document.getElementById("content")
  var config = div.getElementsByClassName("wizard")

  var wizard = new Object()

  for (var i = 0; i < config.length; i++) {

    var name:string
    var value:any
    
    switch (config[i].tagName) {
      case "SELECT":
        name = (config[i] as HTMLSelectElement).name
        value = (config[i] as HTMLSelectElement).value

        // Wenn der Wert eine Zahl ist, wird dieser als Zahl gespeichert
        if(isNaN(value)){
          wizard[name] = value
        } else {
          wizard[name] = parseInt(value)
        }

        break

      case "INPUT":
        switch ((config[i] as HTMLInputElement).type) {
          case "number":
            name = (config[i] as HTMLInputElement).name
            value = parseInt((config[i] as HTMLInputElement).value)

            if (isNaN(value) || value < 1 || value > 100) {
              showToast(name.toUpperCase() + ": " + "{{.alert.missingInput}}", "error")
              return
            }

            wizard[name] = value
            break

          case "text":
            name = (config[i] as HTMLInputElement).name
            value = (config[i] as HTMLInputElement).value

            if (value.length == 0) {
              var msg = name.toUpperCase() + ": " + "{{.alert.missingInput}}"
              showToast(msg, "error")
              return
            }

            wizard[name] = value
            break
        }
        break
      
      default:
        // code...
        break;
    }

  }

  var data = new Object()
  data["wizard"] = wizard

  var server:Server = new Server(cmd)
  server.request(data)

  console.log(data)
}

function finishWizard() {

  var cmd = "saveWizard"
  var data = new Object()
  data["wizard"] = {finish: true}

  var server:Server = new Server(cmd)
  server.request(data)
}

// Wizard
var configurationWizard = new Array()
configurationWizard.push(new WizardItem("tuner", "{{.wizard.tuner.title}}"))
configurationWizard.push(new WizardItem("epgSource", "{{.wizard.epgSource.title}}"))
configurationWizard.push(new WizardItem("m3u", "{{.wizard.m3u.title}}"))
configurationWizard.push(new WizardItem("xmltv", "{{.wizard.xmltv.title}}"))
configurationWizard.push(new WizardItem("webAuth", "{{.wizard.webAuth.title}}"))
configurationWizard.push(new WizardItem("finish", "{{.wizard.finish.title}}"))