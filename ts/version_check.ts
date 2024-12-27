const updateButton = document.getElementById('updateNow') as HTMLButtonElement; // Banner-Element auswählen
const versionModal = document.getElementById('versionModal')
updateButton.addEventListener('click', () => {
    window.open('https://github.com/marcelGoerentz/Threadfin/releases/latest')
})

async function getNewestReleaseFromGithub() {

    await new Promise(resolve => setTimeout(resolve, 1000));
    if (SERVER.clientInfo) {
        if (SERVER.clientInfo.beta) {
            return
        }
        const releasesData = await getReleases();
        if (releasesData) {
            const releases: Release[] = releasesData.map((release: any) => ({
                tag_name: release.tag_name,
            }));

            const currentVersion = parseVersion(SERVER.clientInfo.version);
            const latestReleaseVersion = parseVersion(releases[0].tag_name);

            if (isNewerVersion(latestReleaseVersion, currentVersion)) {
                const modal = new bootstrap.Modal(versionModal)
                modal.show()
            }
        } else {
            console.log('Error fetching releases or no releases found.');
        }
    }
}

function parseVersion(version: string): number[] {
    const regex = /v?(\d+)\.(\d+)(?:\.(\d+))?(?: \((\d+)-\w+\))?/;
    const match = version.match(regex);

    if (match) {
        const major = parseInt(match[1], 10);
        const minor = parseInt(match[2], 10);
        const patch = match[3] ? parseInt(match[3], 10) : (match[4] ? parseInt(match[4], 10) : 0); // Default to 0 if patch is not present
        return [major, minor, patch];
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
    try {
        const response = await fetch('https://api.github.com/repos/marcelGoerentz/Threadfin/releases');
        if (!response.ok) {
            throw new Error(`Error fetching releases. Status: ${response.status}`);
        }
        const releases = await response.json();
        return releases;
    } catch (error) {
        console.error('Error fetching releases:', error);
        return null;
    }
}

interface Release {
    name: string;
    tag_name: string;
    published_at: string;
    prerelease: boolean;
    // Add other relevant properties as needed
}
