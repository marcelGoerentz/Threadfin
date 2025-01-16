class ServerInformation {
    container: HTMLElement;

    constructor() {
        this.container = document.getElementById('server_information');

        const modalDialogue = document.createElement('div');
        modalDialogue.className = 'modal-dialog modal-xl';
        this.container.appendChild(modalDialogue)

        const modalContent = document.createElement('div');
        modalContent.className = 'modal-content';
        modalDialogue.appendChild(modalContent)

        const modalHeader = document.createElement('div');
        modalHeader.className = 'modal-header';
        modalContent.appendChild(modalHeader)


        const modalTitle = document.createElement('h3');
        modalTitle.className = 'modal-title';
        modalTitle.innerHTML = '{{.serverInfo.title}}';
        modalHeader.appendChild(modalTitle);

        const closeButton = document.createElement('button');
        closeButton.className = 'btn-close btn-close-white';
        closeButton.type ='button';
        closeButton.addEventListener('click', () => {
            this.container.style.display = 'none';
            this.container.setAttribute('aria-hidden', 'false')
        })
        closeButton.setAttribute('data-bs-dismiss', 'modal');
        //closeButton.setAttribute('aria-label', 'Close');
        modalHeader.appendChild(closeButton);

        const modalBody = document.createElement('div');
        modalBody.className = 'modal-body';

        const containerFluid = document.createElement('div');
        containerFluid.className = 'container-fluid';

        const row = document.createElement('div');
        row.className = 'row';

        containerFluid.appendChild(row)
        modalBody.appendChild(containerFluid)
        modalContent.appendChild(modalBody)
    }

    addContent(content: ServerInformationGroup[]) {
        const row = this.container.getElementsByClassName('row')[0];
        for (const group of content) {
            row.appendChild(group.GroupHTMLElement);
        }
    }

}

class ServerInformationGroup{
    GroupHTMLElement: HTMLElement
    Header: HTMLElement
    Body: HTMLElement

    constructor(title:string) {
        const group = document.createElement('div');
        group.className = 'card text-bg-dark mb-3';

        const header = document.createElement('div');
        header.className = 'card-header';
        header.innerHTML = title;
        this.Header = header;

        const body = document.createElement('div');
        body.className = 'card-body';
        this.Body = body;

        group.appendChild(header);
        group.appendChild(this.Body);
        this.GroupHTMLElement = group;
    }

    addBodyContent(item:string) {
        switch (item) {
            case 'version':
                this.Body.appendChild(this.addLabel(item, '{{.serverInfo.label.version}}'));
                this.Body.appendChild(this.addTextInput(item, true));
                break;
            case 'errors':
                this.Body.appendChild(this.addLabel(item, '{{.serverInfo.label.errors}}'));
                this.Body.appendChild(this.addTextInput(item, true));
                break;
            case 'warnings':
                this.Body.appendChild(this.addLabel(item, '{{.serverInfo.label.warnings}}'));
                this.Body.appendChild(this.addTextInput(item, true));
                break;
            case 'dvr':
                this.Body.appendChild(this.addLabel(item, '{{.serverInfo.label.dvr}}'));
                this.Body.appendChild(this.addTextInput(item, true));
                break;
            case 'streams':
                this.Body.appendChild(this.addLabel(item, '{{.serverInfo.label.streams}}'));
                this.Body.appendChild(this.addTextInput(item, true));
                break;
            case 'xepg':
                this.Body.appendChild(this.addLabel(item, '{{.serverInfo.label.xepg}}'));
                this.Body.appendChild(this.addTextInput(item, true));
                break;
            case 'm3uUrl':
                this.Body.appendChild(this.addLabel(item, '{{.serverInfo.label.m3uUrl}}'))
                this.Body.appendChild(this.addContainer(item));
                break;
            case 'xepgUrl':
                this.Body.appendChild(this.addLabel(item, '{{.serverInfo.label.xepgUrl}}'))
                this.Body.appendChild(this.addContainer(item));
                break;
            case 'changeVersion':
                this.Body.appendChild(this.addButtonInput(item))
            default:
                console.log('Unknown item: ', item)
        }
    }

    addLabel(forValue:string, text:string): HTMLElement {
        const label = document.createElement('label');
        label.className = 'form-label';
        label.setAttribute('for', forValue);
        label.innerHTML = text;
        return label;
    }

    addTextInput(id:string, disabled:boolean=false):HTMLElement {
        const input = document.createElement('input');
        input.type = 'text';
        input.className = 'form-control';
        input.setAttribute('id', id);
        input.setAttribute('aria-describedby', 'basic-addon3');
        input.readOnly = true;
        if (disabled) {
            input.disabled = true;
        }
        return input;
    }

    addButtonInput(id:string): HTMLElement {
        const input = document.createElement('input');
        input.type = 'button';
        input.setAttribute('id', id);
        return input;
    }

    addContainer(id:string):HTMLElement {
        const container = document.createElement('div');
        container.className ='input-group';
        container.appendChild(this.addTextInput(id));
        container.appendChild(this.addButton(id));
        return container;
    }

    addButton(id:String):HTMLElement {
        const button = document.createElement('button')
        button.type = 'button';
        button.className = 'input-group-text copy-btn';
        button.setAttribute('data-clipboard-target', '#'+ id);
        button.setAttribute('data-bs-toggle', 'tooltip');
        button.setAttribute('data-bs-placement', 'bottom');
        button.setAttribute('data-bs-title', 'Copy to clipboard');
        button.setAttribute('alt', '')
        button.style.backgroundColor = '#333'
        button.style.borderColor = '#444'
        button.innerHTML = '<i class="far fa-clipboard fa-style"></i>';
        return button;
    }
}

class ServerInformationItem extends ServerInformationGroup{

    constructor(groupTitle:string, items:string) {
        super(groupTitle);
        for (const item of items.split(',')) {
            this.addBodyContent(item);
        }
    }
}
