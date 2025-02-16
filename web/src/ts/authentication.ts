function login() {
  let err:Boolean = false
  let data = {}
  let div:any = document.getElementById("content")
  let form:any = document.getElementById("authentication")

  let inputs:any = div.getElementsByTagName("INPUT")

  console.log(inputs)

  for (let i = inputs.length - 1; i >= 0; i--) {
    
    let key:string = (inputs[i] as HTMLInputElement).name
    let value:string = (inputs[i] as HTMLInputElement).value

    if (value.length == 0) {
      inputs[i].style.borderColor = "red"
      err = true
    }

    data[key] = value

  }

  if (err == true) {
    data = {}
    return
  }

  if (data.hasOwnProperty("confirm")) {

    if (data["confirm"] != data["password"]) {
      alert("sdafsd")
      document.getElementById('password').style.borderColor = "red"
      document.getElementById('confirm').style.borderColor = "red"

      document.getElementById("err").innerHTML = "{{.account.failed}}"
      return
    }

  }
  
  console.log(data)

  form.submit();

}