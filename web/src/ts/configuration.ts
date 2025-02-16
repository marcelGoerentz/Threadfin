class WizardCategory {
  DocumentID = "content"

  createCategoryHeadline(value:string):any {
    let element = document.createElement("H4")
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
    let headline = this.createCategoryHeadline(this.headline);
    let key = this.key;
    let content:PopupContent = new PopupContent();
    let description:string;

    let doc = document.getElementById(this.DocumentID);
    doc.innerHTML = "";
    doc.appendChild(headline);

    let text = [];
    let values = [];
    let select: any;
    let input: any;
    switch (key) {
      case "tuner":
        for (let i = 1; i <= 100; i++) {
          text.push(i);
          values.push(i);
        }

        select = content.createSelect(text, values, "1", key);
        select.setAttribute("class", "wizard");
        select.id = key;
        doc.appendChild(select);

        description = "{{.wizard.tuner.description}}";

        break;
      
      case "epgSource":
        text = ["PMS", "XEPG"];
        values = ["PMS", "XEPG"];

        select = content.createSelect(text, values, "XEPG", key);
        select.setAttribute("class", "wizard");
        select.id = key;
        doc.appendChild(select);

        description = "{{.wizard.epgSource.description}}";

        break

      case "m3u":
        input = content.createInput("text", key, "");
        input.setAttribute("placeholder", "{{.wizard.m3u.placeholder}}");
        input.setAttribute("class", "wizard");
        input.id = key;
        doc.appendChild(input);

        description = "{{.wizard.m3u.description}}";

        break

      case "xmltv":
        input = content.createInput("text", key, "");
        input.setAttribute("placeholder", "{{.wizard.xmltv.placeholder}}");
        input.setAttribute("class", "wizard");
        input.id = key;
        doc.appendChild(input);

        description = "{{.wizard.xmltv.description}}";

      break

      default:
        console.log(key);
        break;
    }

    let pre = document.createElement("PRE");
    pre.innerHTML = description;
    doc.appendChild(pre);

    console.log(headline, key);
  }


}


function readyForConfiguration(wizard:number) {

  let server:Server = new Server("getServerConfig");
  server.request({});

  showElement("loading", false);
  configurationWizard[wizard].createWizard()

}

function saveWizard() {

  let cmd = "saveWizard";
  let div = document.getElementById("content");
  let config = div.getElementsByClassName("wizard");

  let wizard = {};

  for (let i = 0; i < config.length; i++) {

    let name:string;
    let value:any;
    
    switch (config[i].tagName) {
      case "SELECT":
        name = (config[i] as HTMLSelectElement).name;
        value = (config[i] as HTMLSelectElement).value;

        // If the value is a number parse it
        if(isNaN(value)){
          wizard[name] = value;
        } else {
          wizard[name] = parseInt(value);
        }

        break

      case "INPUT":
        switch ((config[i] as HTMLInputElement).type) {
          case "text":
            name = (config[i] as HTMLInputElement).name;
            value = (config[i] as HTMLInputElement).value;

            if (value.length == 0) {
              let msg = name.toUpperCase() + ": " + "{{.alert.missingInput}}";
              alert(msg);
              return;
            }

            wizard[name] = value;
            break;
        }
        break;
      
      default:
        // code...
        break;
    }

  }

  let data = {};
  data["wizard"] = wizard;

  let server:Server = new Server(cmd);
  server.request(data);

  console.log(data);
}

// Wizard
var configurationWizard = []
configurationWizard.push(new WizardItem("tuner", "{{.wizard.tuner.title}}"))
configurationWizard.push(new WizardItem("epgSource", "{{.wizard.epgSource.title}}"))
configurationWizard.push(new WizardItem("m3u", "{{.wizard.m3u.title}}"))
configurationWizard.push(new WizardItem("xmltv", "{{.wizard.xmltv.title}}"))