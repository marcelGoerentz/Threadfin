async function getNewestReleaseFromGithub() {

    await new Promise(resolve => setTimeout(resolve, 1000));
    if (SERVER.clientInfo) {
        /*if (SERVER.clientInfo.beta) {
            return
        }*/
        const releasesData = await getReleases();
        if (releasesData) {
            const releases: Release[] = releasesData.map((release: any) => ({
                tag_name: release.tag_name,
                prerelease: release.prerelease,
            }));

            let currentVersion: number[]
            try{
                 currentVersion = parseVersion(SERVER.clientInfo.version);
            } catch (error: any) {
                console.log("Unable to parse current version info!")
                return
            }
            let latestAvailableVersion: Release;
            if (SERVER.clientInfo.beta) {
                latestAvailableVersion = releases.find(release => release.prerelease == true);
            } else {
                latestAvailableVersion = releases.find(release => release.prerelease == false);
            }
            let latestReleaseVersion: number[];
            try {
             latestReleaseVersion = parseVersion(latestAvailableVersion.tag_name);
            } catch (error: any) {
                console.log("Unable to parse GitHub version info!")
                return
            }

            if (isNewerVersion(latestReleaseVersion, currentVersion)) {
                const notification_container = document.getElementById('notification_container') as HTMLElement;
                const notificationTitle = document.getElementById('notification_title') as HTMLElement;
                const notificationText = document.getElementById('notification_text') as HTMLElement;

                notificationTitle.innerHTML = '{{.notification.update.title}}';
                notificationText.innerHTML = '{{.notification.update.content}}';

                const closeButton = document.getElementById('closeNotification') as HTMLButtonElement;
                const updateButton = document.getElementById('updateNowButton') as HTMLButtonElement;
                updateButton.value = '{{.button.update}}';
                closeButton.onclick = () => {
                    notification_container.style.display = 'none';
                }
                updateButton.onclick = () => {
                    updateButton.value = '{{.button.updating}}';
                    const server: Server = new Server("updateThreadfin")
                    server.request({})
                    setTimeout(() => {
                        location.reload()
                    }, 20000);
                }
                notification_container.style.display = 'block';
            }
        } else {
            console.log('Error fetching releases or no releases found.');
        }
    }
}

function parseVersion(version: string): number[]  {
    // TODO: Improve version parsing
    const regex = /^v?(\d+)\.(\d+)(?:\.(\d+))?(?:\.(\d+))?(?: \((\d+)(?:-(\w+))?\))?(?:-(\w+))?$/;
    const match = version.match(regex);

    if (match) {
        const major = parseInt(match[1], 10);
        const minor = parseInt(match[2], 10);
        const patch = match[3] ? parseInt(match[3], 10) : (match[4] ? parseInt(match[4], 10) : 0); // Default to 0 if patch is not present
        const build = match[4] ? parseInt(match[4], 10) : (match[5] ? parseInt(match[5], 10) : 0); // Default to 0 if patch is not present
        return [major, minor, patch, build];
    } else {
        throw new Error("Invalid version format");
    }
}

function isNewerVersion(latest: number[], current: number[]): boolean {
    for (let i = 0; i < latest.length; i++) {
        if (latest[i] > current[i]) return true;
        if (latest[i] < current[i]) return false;
    }
    return false;
}

async function getReleases(): Promise<any> {
    const response = await fetch('https://api.github.com/repos/marcelGoerentz/Threadfin/releases');
    if (response.ok) {
        return await response.json();
    } else {
        console.error(`Error fetching releases: Status: ${response.status}`);
        return null;
    }
}

// Define the Release interface
interface Release {
    name: string;
    tag_name: string;
    published_at: string;
    prerelease: boolean;
    // Add other relevant properties as needed
}
