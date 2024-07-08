package imgcache

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

type ImageCache struct {
	cache map[string]string
	caching bool
	basePath string
	baseURL string
	httpPool *sync.Pool
	mutex sync.Mutex
	wg sync.WaitGroup
}

// Create a new image cache
func NewImageCache(caching bool, basePath string, baseURL string) (*ImageCache) {
	return &ImageCache{
		caching: caching,
		basePath: basePath,
		baseURL: baseURL,
		cache: make(map[string]string),
		httpPool: &sync.Pool{
			New: func() interface {} {
				return &http.Client{}
			},
		},
	}
}

func (ic *ImageCache) UpdateBaseURL(url string) {
	ic.baseURL = url
}

// Enqueue the URL for downloading
func (ic *ImageCache) EnqueueURL(url string, filename string) {
	ic.wg.Add(1)
	go func(url string) {
		defer ic.wg.Done()
		ic.DownloadImage(url, filename)
	}(url)
}

// Get the Url to the Image cached or original
func (ic *ImageCache) GetImageURL(url string) (string) {
	// Generate the key from the URL
	key := createKeyFromUrl(url)
	// If image is already cached return the url
	ic.mutex.Lock()
	if cached_url, ok := ic.cache[key]; ok {
		ic.mutex.Unlock()
		return cached_url
	}
	ic.mutex.Unlock()

	if ic.caching {
		filename := createFileNameFromURL(url, key)
		path_to_file := ic.basePath + filename
		url_to_file := ic.baseURL + "/cache/" + filename

		// Enqueue the Image for the download
		ic.EnqueueURL(url, path_to_file)

		// Save file name in cache
		ic.mutex.Lock()
		ic.cache[key] = url_to_file
		ic.mutex.Unlock()
		return url_to_file
	} else {
		// Save original url in cache
		return url
	}
}

// Download the Image
func (ic *ImageCache) DownloadImage(url string, filename string) (error) {

	// Check if file already exists
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		// Get a HTTP-Connection from the pool
		client := ic.httpPool.Get().(*http.Client)
		defer ic.httpPool.Put(client)

		// Download the image
		resp, err := client.Get(url)
		if err != nil {
			ic.ErrorHandlingWhenDownloading(url)
			return errors.New("error when downloading the image")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == 404 {
				url = "https://" + strings.Split(url, "//")[1]
				resp, err = client.Get(url)
				if err != nil {
					ic.ErrorHandlingWhenDownloading(url)
					return errors.New("error when downloading the image")
				}
				if resp.StatusCode != http.StatusOK {
					ic.ErrorHandlingWhenDownloading(url)
					return errors.New("received bad status code")
				}
			} else {
				ic.ErrorHandlingWhenDownloading(url)
				return errors.New("received bad status code")
			}
		}

		// Save the image to disk
		file, err := os.Create(filename)
		if err != nil {
			ic.ErrorHandlingWhenDownloading(url)
			return errors.New("unable to create the file")
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			ic.ErrorHandlingWhenDownloading(url)
			return errors.New("can't save the image to the file")
		}
		return nil
	}
	return nil
}

func (ic *ImageCache) ErrorHandlingWhenDownloading(url string) {
	ic.mutex.Lock()
	defer ic.mutex.Unlock()
	ic.cache[url]=url
}

// Block until downloads have been completed
func (ic *ImageCache) WaitForDownloads() {
	ic.wg.Wait()
}

func (ic *ImageCache) GetNumCachedImages() int {
    ic.mutex.Lock()
    defer ic.mutex.Unlock()
    return len(ic.cache)
}

// Clear the cache but not the files
func (ic *ImageCache) DeleteCache() {
    if ic.caching {
        ic.cache = make(map[string]string) // Clear the cache
    }
}
