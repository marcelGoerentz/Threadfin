const bannerElement = document.querySelector('.banner') as HTMLElement; // Banner-Element auswählen

async function getNewestReleaseFromGithub() {

    const releasesData = await getReleases();
    if (releasesData) {
        const releases: Release[] = releasesData.map((release: any) => ({
            tag_name: release.tag_name,
        }));
        // Get tag name
        var release_tag = releases[0]["tag_name"];
        const split = release_tag.split(".");
        const release_major_version = split[0][1]
        const release_minor_version = split[1]
        const release_build_version = split[2]
        const release_version = [release_major_version, release_minor_version, release_build_version]
        let current_version = []
        
        if ('clientInfo' in SERVER) {
            var current_version_string = SERVER["clientInfo"]["version"];
            current_version.push(current_version_string.split(".")[0]);
            current_version.push(current_version_string.split(".")[1][0]);
            current_version.push(current_version_string.split("(")[1][0]);
        }
        if (current_version.length !== 0) {
            for (let i = 0; i < 3; i++) {
                if (release_version[i] > current_version[i]) {
                    bannerElement.innerHTML = 'New Version available! Click <a href="https://github.com/marcelGoerentz/Threadfin/releases/latest">here</a> to download.';
                    bannerElement.style.display = 'block'; // Show Banner if newer version is available
                    break
                } else if (release_version[i] < current_version[i]) {
                    break
                }
            }
        }
    } else {
        console.log('Error fetching releases or no releases found.');
    }
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
    // Add other relevant properties as needed
}

