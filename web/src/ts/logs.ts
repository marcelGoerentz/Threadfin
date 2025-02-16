class Log {

  createLog(entry:string):any {

    let element = document.createElement("PRE");

    if (entry.indexOf("WARNING") != -1) {
      element.className = "warningMsg";
    }

    if (entry.indexOf("ERROR") != -1) {
      element.className = "errorMsg";
    }

    if (entry.indexOf("DEBUG") != -1) {
      element.className = "debugMsg";
    }

    element.innerHTML = entry;

    return element;
  }

}

function showLogs(bottom:boolean) {

  let log = new Log();

  let logs = SERVER["log"]["log"];
  let div = document.getElementById("content_log");

  div.innerHTML = ""

  let keys = getObjKeys(logs);

  keys.forEach(logID => {

    let entry = log.createLog(logs[logID]);

    div.append(entry);
  
  });

  setTimeout(function(){ 

    if (bottom == true) {
  
      let wrapper = document.getElementById("box-wrapper");
      wrapper.scrollTop = wrapper.scrollHeight;

    }

  }, 10);

}

function resetLogs() {

  let cmd = "resetLogs";
  let data = {};
  let server:Server = new Server(cmd);
  server.request(data)

}