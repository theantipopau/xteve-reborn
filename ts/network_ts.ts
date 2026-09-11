class Server {
  protocol:string
  cmd:string

  constructor(cmd:string) {
    this.cmd = cmd
  }

  request(data:Object):any {

    // The periodic background log poll (every 10s, see menu_ts.ts) used to
    // share the same SERVER_CONNECTION lock as every user-initiated action.
    // Any click that landed while a poll was in flight was silently
    // dropped - no error, no feedback, just an unresponsive button until
    // the next poll cycle happened to leave a gap. It gets its own lock so
    // it can never block real user interactions.
    var isBackgroundPoll = (this.cmd == "updateLog")

    if (isBackgroundPoll) {
      if (LOG_POLL_CONNECTION == true) {
        return
      }
      LOG_POLL_CONNECTION = true
    } else {
      if (SERVER_CONNECTION == true) {
        return
      }
      SERVER_CONNECTION = true
    }

    console.log(data)
    if (this.cmd != "updateLog") {
      showElement("loading", true)
      UNDO = new Object()
    }

    switch(window.location.protocol) {
      case "http:":
        this.protocol = "ws://"
        break
      case "https:":
        this.protocol = "wss://"
        break
    }

    var url = this.protocol + window.location.hostname + ":" + window.location.port + "/data/" + "?Token=" + getCookie("Token")

    data["cmd"] = this.cmd
    var ws = new WebSocket(url)

    // A request that never gets a response (dropped connection, server
    // restart mid-request, flaky network) used to leave SERVER_CONNECTION
    // stuck at true forever, silently freezing every subsequent click in
    // the UI with no explanation until the page was reloaded. This timeout
    // guarantees the lock is released and the user is told what happened.
    var settled = false
    var timeout = window.setTimeout(function() {

      if (settled == true) {
        return
      }

      settled = true
      if (isBackgroundPoll) {
        LOG_POLL_CONNECTION = false
      } else {
        SERVER_CONNECTION = false
        showToast("xTeVe did not respond in time. Please try again.", "error")
      }
      showElement("loading", false)
      ws.close()

    }, 15000)

    ws.onopen = function() {

      WS_AVAILABLE = true

      console.log("REQUEST (JS):");
      console.log(data)

      console.log("REQUEST: (JSON)");
      console.log(JSON.stringify(data))

      this.send(JSON.stringify(data));

    }

    ws.onerror = function(e) {

      if (settled == true) {
        return
      }
      settled = true
      window.clearTimeout(timeout)

      console.log("No websocket connection to xTeVe could be established. Check your network configuration.")

      if (isBackgroundPoll) {
        LOG_POLL_CONNECTION = false
      } else {
        SERVER_CONNECTION = false
      }
      showElement("loading", false)

      if (WS_AVAILABLE == false && isBackgroundPoll == false) {
        showToast("No websocket connection to xTeVe could be established. Check your network configuration.", "error")
      }

    }


    ws.onmessage = function (e) {

      if (settled == true) {
        return
      }
      settled = true
      window.clearTimeout(timeout)

      if (isBackgroundPoll) {
        LOG_POLL_CONNECTION = false
      } else {
        SERVER_CONNECTION = false
      }
      showElement("loading", false)

      console.log("RESPONSE:");
      var response = JSON.parse(e.data);
  
      console.log(response);

      if (response.hasOwnProperty("token")) {
        document.cookie = "Token=" + response["token"]
      }

      if (response["status"] == false) {

        showToast(response["err"], "error")

        if (response.hasOwnProperty("reload")) {
          // A toast isn't blocking like the alert() it replaced, so an
          // immediate reload would wipe it before it's readable - give it a
          // moment on screen first.
          setTimeout(function() { location.reload() }, 1500)
        }

        return
      }


      if (response.hasOwnProperty("logoURL")) {
        var div = (document.getElementById("channel-icon") as HTMLInputElement)
        div.value = response["logoURL"]
        div.className = "changed"
        return
      }

      switch (data["cmd"]) {
        case "updateLog":
          SERVER["log"] = response["log"]
          if (document.getElementById("content_log")) {
            showLogs(false)
          }
          return
          break;
        
        default:
          SERVER = new Object()
          SERVER = response
          break;
      }

      if (response.hasOwnProperty("openMenu")) {
        var menu = document.getElementById(response["openMenu"])
        menu.click()
        showElement("popup", false)
      }

      if (response.hasOwnProperty("openLink")) {
        window.location = response["openLink"]
      }

      var alertShown = false
      if (response.hasOwnProperty("alert")) {
        showToast(response["alert"], "info")
        alertShown = true
      }

      if (response.hasOwnProperty("reload")) {
        setTimeout(function() { location.reload() }, alertShown ? 1500 : 0)
      }


      if (response.hasOwnProperty("wizard")) {
        createLayout()
        configurationWizard[response["wizard"]].createWizard()
        return
      }

      createLayout()

    }
  
  }
  
}

function getCookie(name) {
  var value = "; " + document.cookie;
  var parts = value.split("; " + name + "=");
  if (parts.length == 2) return parts.pop().split(";").shift();
}