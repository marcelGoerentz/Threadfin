const bannerElement = document.querySelector('.banner') as HTMLElement; // Banner-Element auswählen

async function getNewestReleaseFromGithub() {

    const releasesData = await getReleases();
    if (releasesData) {
        const releases: Release[] = releasesData.map((release: any) => ({
            tag_name: release.tag_name,
        }));
        // Get tag name
        var release_tag = releases[0]["tag_name"];
        const regex = /[^\d]/gi;
        // Create Number from tag name
        const latest_version = Number(release_tag.replace(regex, ''));
        const version_elemnt = document.getElementById('version') as HTMLInputElement;
        const current_version = Number(version_elemnt.value.replace(regex, ''));
        if (latest_version > current_version) {
            bannerElement.style.display = 'block'; // Show Banner if newer version is available
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

