// Function to create the option Dialogue
function showIPBindingDialogue() {
  var myModal = new bootstrap.Modal(document.getElementById('popup'));
  const fragment = document.createDocumentFragment();
  const popupModalContent = document.getElementById('popupModalContent');
  fragment.appendChild(popupModalContent);
  editCustomPopUpContainer(fragment);
  addCustomPopUpContent(fragment);

  const parent = document.getElementById('popup');
  const child = parent.children[0].appendChild(fragment);
  showElement("popup", true);
}

function editCustomPopUpContainer(fragment: DocumentFragment){
  const popupHeader = fragment.getElementById('popupHeader');
  const popupRow = fragment.getElementById('popupRow')
  const popupCustom = fragment.getElementById('popupCustom');

  // Remove former header and add a custom one
  const h3 = popupHeader.querySelector('h3')
  if (h3) {
    popupHeader.removeChild(h3)
  }
  
  const headline = createElementWithAttributes('h3', {
    class: 'modal-title',
    id: 'modalHeadline'
  });
  headline.textContent = 'IP selection';
  popupHeader.appendChild(headline);

  const xButton = createElementWithAttributes('button', {
    type: 'button',
    class: 'btn-close'
  });
  popupHeader.appendChild(xButton);

  // Delete and renew popupCustom
  popupCustom.remove()
  const newPopupCustom = createElementWithAttributes('div', {id: 'popupCustom'});

  popupRow.appendChild(newPopupCustom);
  const table = createElementWithAttributes('table', {
    id: 'optionTable'
  });
  newPopupCustom.appendChild(table);

}

function addCustomPopUpContent(fragment: DocumentFragment) {
  const content: PopupContent = new PopupContent();

  if ("clientInfo" in SERVER) {    
    const bindingIPsElement = document.getElementById('bindingIPs') as HTMLInputElement;
    const bindingIPs: string = bindingIPsElement.getAttribute('value');
    const bindingIPspArray = bindingIPs.split(";");
    const systemnIPs: Array<string> = SERVER["clientInfo"]["systemIPs"];
    const optionTable = fragment.getElementById('optionTable');
    
    systemnIPs.forEach((ipAddress, index) => {
      if (!ipAddress.includes('169.254')) {
        const tr = document.createElement('tr');
        const tdLeft = document.createElement('td');
        const tdRight = document.createElement('td');

        const checkbox = createCheckbox(ipAddress, 'ipCheckbox' + index);
        checkbox.checked = bindingIPspArray.includes(ipAddress);

        const label = document.createElement("label");
        label.setAttribute("for", "ipCheckbox" + index);
        label.innerHTML = ipAddress;

        tdLeft.appendChild(checkbox);
        tdRight.appendChild(label);
        tr.appendChild(tdLeft);
        tr.appendChild(tdRight);
        optionTable.appendChild(tr);
      }
    });

    const checkbox_container = fragment.getElementById('popupCustom');
    checkbox_container.textContent = 'Select one or more IP(s). If none has been selected then Threadfin will bind to all of them!'; // This deletes all nodes and replace with text!
    checkbox_container.appendChild(optionTable); // Reappend the table

    const saveButton = createButton(content, "buttonUpdate", "{{.button.update}}", 'javascript: updateBindingIPs()');
    const cancelButton = createButton(content, "buttonCancel", "{{.button.cancel}}", 'javascript: resetPopup()');
    checkbox_container.appendChild(saveButton);
    checkbox_container.appendChild(cancelButton);
    
    const ipSelection = fragment.getElementById('popupHeader');
    const closeButton = ipSelection.querySelector('button.btn-close');
    closeButton.addEventListener('click', () => resetPopup());
  }
}

function createButton(content: PopupContent, id: string, text: string, onClick?: string): HTMLInputElement {
  return createInput('button', id, text, {'onclick': onClick}) as HTMLInputElement
}

function createCheckbox(name: string, id: string = ''): HTMLInputElement {
  return createInput('checkbox', name, {}, {id: id}) as HTMLInputElement
}

function resetPopup() {

  // remove cancel x-button and headline from popupHeader
  const popupHeader = document.getElementById('popupHeader');
  popupHeader.removeChild(popupHeader.querySelector('button'))
  popupHeader.removeChild(popupHeader.querySelector('h3'))

  // remove existing popupCustom
  const optionsDialogue = document.getElementById('popupCustom');
  optionsDialogue.remove();

  // add new popup
  const popupRow = document.getElementById('popupRow');
  const newPopupCustom = createElementWithAttributes('div', {
    id: 'popupCustom'
  })
  popupRow.appendChild(newPopupCustom)

  // don't show the popup anymore
  showElement('popup', false)
}

function updateBindingIPs() {
  const checkboxTable = document.getElementById('optionTable');
  const checkboxList = checkboxTable.querySelectorAll('input[type="checkbox"]');
  // get checked boxes and create array
  var bindingIPs: string[] = Array.from(checkboxList)
    .filter(checkbox => (checkbox as HTMLInputElement).checked)
    .map(checkbox => (checkbox as HTMLInputElement).name);
  
  const bindingIPsElement = document.getElementById('bindingIPs');
  if (bindingIPs.length === 0) {
    // set value to none
    bindingIPsElement.setAttribute('value', '')
  } else {
    // insert the values from the array
    bindingIPsElement.setAttribute('value', bindingIPs.join(';') + ";");
  }
  // tell about the change
  bindingIPsElement.setAttribute('class', 'changed');
  // resetPopup for the next run
  resetPopup()
}

function createInput(type, name, value, attribute = {}) {
  return createElementWithAttributes('input', { type, name, value, ...attribute });
}

function createElementWithAttributes(tag: string, attributes: object) {
  const element = document.createElement(tag);
  for (const key in attributes) {
    element.setAttribute(key, attributes[key]);
  }
  return element;
}