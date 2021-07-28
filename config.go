package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

func buildDefaultConfig() error {
	content, e := comicBook.List()
	if e != nil {
		log.Fatal(e)
	}
	for _, fileName := range content {
		ext := strings.ToLower(fileName)
		if strings.HasPrefix(fileName, "PDF Page") || strings.HasSuffix(ext, ".jpg") || strings.HasSuffix(ext, ".jpeg") || strings.HasSuffix(ext, ".webp") || strings.HasSuffix(ext, ".png") || strings.HasSuffix(ext, ".gif") {
			album.Pages = append(album.Pages, PageData{
				FileName: fileName,
				Visible:  true,
			})
		}
	}
	if len(album.Pages) == 0 {
		log.Fatal("No image found in comicBook")
	}

	sort.Slice(album.Pages, func(i, j int) bool {
		return album.Pages[i].FileName < album.Pages[j].FileName
	})

	return nil
}

func getGlobalConfigurationFile() string {
	return path.Join(configFolder, "config.yml")
}

func readConfiguration(fileMD5 string) error {
	var err error

	globalConfigurationFile := getGlobalConfigurationFile()
	_, err = os.Stat(globalConfigurationFile)
	if err == nil {
		log.Printf("Loading global configuration from %s\n", globalConfigurationFile)
		fileData, err := ioutil.ReadFile(globalConfigurationFile)
		if err != nil {
			return err
		}

		err = yaml.Unmarshal(fileData, &preferences)
		if err != nil {
			panic(err)
		}
	}
	if preferences.WindowedSize.w == 0 {
		preferences.WindowedSize = Size{
			w: 600,
			h: 800,
		}
	}

	album.Pages = make([]PageData, 0)
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
			return err
		}

		err = yaml.Unmarshal(fileData, &album)
		if err != nil {
			panic(err)
		}
	}

	log.Printf("Album has %d pages\n", len(album.Pages))

	return err
}

func saveConfiguration() error {
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
