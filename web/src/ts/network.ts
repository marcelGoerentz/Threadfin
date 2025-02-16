class Server {
  protocol: string
  cmd: string

  constructor(cmd: string) {
    this.cmd = cmd
  }

  request(data: Object): any {

    //if (SERVER_CONNECTION == true) {
    //  return
    //}

    SERVER_CONNECTION = true;

    console.log(data);
    if (this.cmd != "updateLog") {
      // showElement("loading", true)
      UNDO = {};
    }

    switch (window.location.protocol) {
      case "http:":
        this.protocol = "ws://";
        break;
      case "https:":
        this.protocol = "wss://";
        break;
    }

    let url = this.protocol + window.location.hostname + ":" + window.location.port + "/ws/" + "?Token=" + getCookie("Token");

    data["cmd"] = this.cmd;
    let ws = new WebSocket(url);
    ws.onopen = function () {

      WS_AVAILABLE = true;

      console.log("REQUEST (JS):");
      console.log(data);

      console.log("REQUEST: (JSON)");
      console.log(JSON.stringify(data));

      this.send(JSON.stringify(data));

    }

    ws.onerror = function () {

      console.log("No websocket connection to Threadfin could be established. Check your network configuration.");
      SERVER_CONNECTION = false;

      if (WS_AVAILABLE == false) {
        alert("No websocket connection to Threadfin could be established. Check your network configuration.");
      }

    }


    ws.onmessage = function (e) {

      SERVER_CONNECTION = false;
      showElement("loading", false);

      console.log("RESPONSE:");
      let response = JSON.parse(e.data);

      console.log(response);

      if (response.hasOwnProperty("token")) {
        document.cookie = "Token=" + response["token"];
      }
      
      if (response.error) {
        console.log(response.error);
        return;
      }


      if (response.hasOwnProperty("logoURL")) {
        let div = (document.getElementById("channel-icon") as HTMLInputElement);
        div.value = response["logoURL"];
        div.className = "changed";
        return;
      }

      switch (data["cmd"]) {
        case "updateLog":
          SERVER.log = response["log"];
          if (document.getElementById("content_log")) {
            showLogs(false);
          }
          return;

        default:
          SERVER = response;
          break;
      }

      if (response.hasOwnProperty("openMenu")) {
        let menu = document.getElementById(response["openMenu"]);
        menu.click();
        showElement("popup", false);
      }

      if (response.hasOwnProperty("openLink")) {
        window.location = response["openLink"];
      }

      if (response.hasOwnProperty("alert")) {
        alert(response["alert"]);
      }

      if (response.hasOwnProperty("reload")) {
        location.reload();
      }


      if (response.hasOwnProperty("wizard")) {
        createLayout();
        configurationWizard[response["wizard"]].createWizard();
        return;
      }

      createLayout();

    }

  }

}

function getCookie(name: string) {
  let value = "; " + document.cookie;
  let parts = value.split("; " + name + "=");
  if (parts.length == 2) return parts.pop().split(";").shift();
}
