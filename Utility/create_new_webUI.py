import os
import base64

# Variablen
html_folder = "./web/public"
go_file = "./src/webUI.go"
package_name = "src"
map_name = "webUI"

# Funktion zum Überprüfen der HTML-Dateien
def check_html_folder():
    if not os.path.isdir(html_folder):
        print("HTML folder does not exist.")
        return False
    return True

# Funktion zum Erstellen der Go-Datei
def build_go_file():
    if not check_html_folder():
        return

    content = f"package {package_name}\n\n"
    content += f"var {map_name} = make(map[string]interface{{}})\n\n"
    content += "func loadHTMLMap() {\n\n"

    # Hier wird die Map aus den Dateien im HTML-Ordner erstellt
    content += create_map_from_files(html_folder) + "\n"

    content += "}\n\n"

    with open(go_file, "w") as f:
        f.write(content)

# Funktion zum Erstellen der Map aus den Dateien im HTML-Ordner
def create_map_from_files(folder):
    map_content = ""
    for root, _, files in os.walk(folder):
        for file in files:
            file_path = os.path.join(root, file)
            with open(file_path, "rb") as f:
                base64_str = base64.b64encode(f.read()).decode('utf-8')
            key = os.path.relpath(file_path, folder).replace("\\", "/")
            map_content += f'\t{map_name}["web/public/{key}"] = "{base64_str}"\n'
    return map_content

# Aufruf der Funktion buildGoFile
build_go_file()
print("Created new webUI.go")