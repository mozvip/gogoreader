package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/faiface/pixel"
	"gopkg.in/yaml.v3"
)

func buildDefaultConfig() error {
	content, e := comicBook.List()
	if e != nil {
		log.Fatal(e)
	}
	for _, fileName := range content {
		if strings.HasPrefix(fileName, "__MACOSX") {
			continue
		}
		ext := strings.ToLower(fileName)
		if strings.HasPrefix(fileName, "PDF Page") || strings.HasSuffix(ext, ".jpg") || strings.HasSuffix(ext, ".jpeg") || strings.HasSuffix(ext, ".webp") || strings.HasSuffix(ext, ".png") || strings.HasSuffix(ext, ".gif") {
			album.Images = append(album.Images, &ImageData{
				FileName: fileName,
				Visible:  true,
			})
		}
	}
	if len(album.Images) == 0 {
		return errors.New("no image found in comicBook")
	}

	// sort images by their filename
	sort.Slice(album.Images, func(i, j int) bool {
		return album.Images[i].FileName < album.Images[j].FileName
	})

	// create a default page for each of these images
	album.Pages = make([]*PageData, len(album.Images))
	for i, img := range album.Images {
		album.Pages[i] = &PageData{Images: []*ImageData{img}}
	}

	return nil
}

func readConfiguration(fileMD5 string) (Preferences, error) {
	var err error
	var preferences = NewPreferences()

	globalConfigurationFile := getGlobalConfigurationFile()
	_, err = os.Stat(globalConfigurationFile)
	if err == nil {
		log.Printf("Loading global configuration from %s\n", globalConfigurationFile)
		fileData, err := ioutil.ReadFile(globalConfigurationFile)
		if err != nil {
			return preferences, err
		}
		err = yaml.Unmarshal(fileData, &preferences)
		if err != nil {
			panic(err)
		}
	}
	if preferences.WindowedSize.X == 0 {
		preferences.WindowedSize = pixel.Vec{
			X: 800,
			Y: 600,
		}
	}

	album.Pages = make([]*PageData, 0)
	album.MD5 = fileMD5

	configurationFile := album.GetConfigurationFile(configFolder)
	_, err = os.Stat(configurationFile)
	if os.IsNotExist(err) {
		log.Printf("%s was not found, initializing default config\n", configurationFile)
		err = buildDefaultConfig()
	} else {
		log.Printf("Loading configuration from %s\n", configurationFile)
		fileData, err := ioutil.ReadFile(configurationFile)
		if err != nil {
			return preferences, err
		}

		err = yaml.Unmarshal(fileData, &album)
		if err != nil {
			panic(err)
		}
	}

	log.Printf("Album has %d pages\n", len(album.Pages))

	return preferences, err
}

func saveConfiguration(preferences Preferences) error {
	d, err := yaml.Marshal(&album)
	if err != nil {
		return err
	}
	var configFile = album.GetConfigurationFile(configFolder)
	log.Printf("Saving comicBook configuration to %s\n", configFile)
	err = ioutil.WriteFile(configFile, d, 0644)
	if err != nil {
		return err
	}

	prefs, err := yaml.Marshal(&preferences)
	if err != nil {
		return err
	}
	log.Printf("Saving global configuration to %s\n", getGlobalConfigurationFile())
	err = ioutil.WriteFile(getGlobalConfigurationFile(), prefs, 0644)
	if err != nil {
		return err
	}

	return nil
}
