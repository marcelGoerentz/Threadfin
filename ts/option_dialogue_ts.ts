// Function to create the option Dialogue
function createOptionDialogueContainer() {
  const fragment = document.createDocumentFragment();

  const optionsDialogue = createElementWithAttributes('div', {
    class: 'modal fade',
    id: 'dialogueContainer'
  });

  const modalDialog = createElementWithAttributes('div', {
    class: 'modal-dialog modal-xl'
  });
  optionsDialogue.appendChild(modalDialog);

  const modalContent = createElementWithAttributes('div', {
    class: 'modal-content'
  });
  modalDialog.appendChild(modalContent);

  const modalHeader = createElementWithAttributes('div', {
    class: 'modal-header'
  });
  modalContent.appendChild(modalHeader);

  const headline = createElementWithAttributes('h3', {
    class: 'modal-title',
    id: 'optionHeadline'
  });
  modalHeader.appendChild(headline);

  const xButton = createElementWithAttributes('button', {
    type: 'button',
    class: 'btn-close',
    'data-bs-dismiss': 'modal',
    'aria-label': 'Close'
  });
  modalHeader.appendChild(xButton);

  const modalBody = createElementWithAttributes('div', {
    class: 'modal-body'
  });
  modalContent.appendChild(modalBody);

  const fluidContainer = createElementWithAttributes('div', {
    class: 'container-fluid'
  });
  modalBody.appendChild(fluidContainer);

  const row = createElementWithAttributes('div', {
    class: 'row'
  });
  fluidContainer.appendChild(row);

  const card = createElementWithAttributes('div', {
    class: 'card text-bg-dark mb-3'
  });
  row.appendChild(card);

  const cardBody = createElementWithAttributes('div', {
    class: 'card-body',
    id: 'optionCardBody'
  });
  card.appendChild(cardBody);

  const table = createElementWithAttributes('table', {
    id: 'optionTable'
  });
  cardBody.appendChild(table);

  fragment.appendChild(optionsDialogue);
  document.body.appendChild(fragment);
}

function createBindingIPsOptionDialogue() {
  const content: PopupContent = new PopupContent();
  const fragment = document.createDocumentFragment();

  if ("clientInfo" in SERVER) {
    const optionHeadline = document.getElementById('optionHeadline');
    optionHeadline.textContent = 'IP selection';
    const bindingIPsElement = document.getElementById('bindingIPs') as HTMLInputElement;
    const bindingIPs: string = bindingIPsElement.getAttribute('value');
    const bindingIPspArray = bindingIPs.split(";");
    const systemnIPs: Array<string> = SERVER["clientInfo"]["systemIPs"];
    
    systemnIPs.forEach((ipAddress, index) => {
      if (!ipAddress.includes('169.254')) {
        const tr = document.createElement('tr');
        const tdLeft = document.createElement('td');
        const tdRight = document.createElement('td');

        const checkbox = content.createCheckbox(ipAddress, 'ipCheckbox' + index);
        checkbox.checked = bindingIPspArray.includes(ipAddress);

        const label = document.createElement("label");
        label.setAttribute("for", "ipCheckbox" + index);
        label.innerHTML = ipAddress;

        tdLeft.appendChild(checkbox);
        tdRight.appendChild(label);
        tr.appendChild(tdLeft);
        tr.appendChild(tdRight);
        fragment.appendChild(tr);
      }
    });

    const optionTable = document.getElementById("optionTable");
    optionTable.appendChild(fragment);

    const checkbox_container = document.getElementById('optionCardBody');
    checkbox_container.textContent = 'Select one or more IP(s). If none has been selected then Threadfin will bind to all of them!'; // This deletes all nodes and replace with text!
    checkbox_container.appendChild(optionTable);
    const saveButton = createButton(content, "buttonUpdate", "{{.button.update}}", 'javascript: updateBindingIPs()');
    const cancelButton = createButton(content, "buttonCancel", "{{.button.cancel}}");
    checkbox_container.appendChild(saveButton);
    checkbox_container.appendChild(cancelButton);

    
    
    
    const ipSelection = document.getElementById('dialogueContainer');
    const closeButton = ipSelection.querySelector('button.btn-close');
    closeButton.addEventListener('click', () => resetOptionsDialogue());
    cancelButton.addEventListener('click', () => resetOptionsDialogue());
  }
}

function createButton(content: PopupContent, id: string, text: string, onClick?: string): HTMLInputElement {
  const button = content.createInput("button", id, text);
  if (onClick) {
    button.setAttribute("onclick", onClick);
  }
  button.setAttribute('data-bs-target', '#dialogueContainer');
  button.setAttribute("data-bs-toggle", "modal");
  return button;
}

function resetOptionsDialogue() {
  let optionsDialogue = document.getElementById('dialogueContainer');
  
  if (optionsDialogue) {
    // Remove the existing modal
    optionsDialogue.remove();
  }

  // Create a new modal
  createOptionDialogueContainer();
}

function updateBindingIPs() {
  const checkboxTable = document.getElementById('optionTable');
  const checkboxList = checkboxTable.querySelectorAll('input[type="checkbox"]');
  var bindingIPs: string[] = Array.from(checkboxList)
    .filter(checkbox => (checkbox as HTMLInputElement).checked)
    .map(checkbox => (checkbox as HTMLInputElement).name);
  const bindingIPsElement = document.getElementById('bindingIPs');
  if (bindingIPs.length === 0) {
    bindingIPsElement.setAttribute('value', '')
  } else {
    bindingIPsElement.setAttribute('value', bindingIPs.join(';') + ";");
  }
  bindingIPsElement.setAttribute('class', 'changed');
  resetOptionsDialogue()
}

function createElementWithAttributes(tag: string, attributes: object) {
  const element = document.createElement(tag);
  var test = {id: 'test'}
  for (const key in attributes) {
    element.setAttribute(key, attributes[key]);
  }
  return element;
}