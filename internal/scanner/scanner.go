package scanner

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ModDescriptor struct {
	Version string `xml:"version"`
	Author  string `xml:"author"`
}

type modDesc struct {
	XMLName xml.Name `xml:"modDesc"`
	Version string   `xml:"version"`
	Author  string   `xml:"author"`
}

func ScanLocalMods(directory string) (map[string]ModDescriptor, error) {
	result := make(map[string]ModDescriptor)

	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".zip") {
			continue
		}

		zipPath := filepath.Join(directory, name)
		desc, err := extractModDesc(zipPath)
		if err != nil {
			result[name] = ModDescriptor{Version: "unknown"}
			continue
		}

		result[name] = desc
	}

	return result, nil
}

func extractModDesc(zipPath string) (ModDescriptor, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return ModDescriptor{}, err
	}
	defer reader.Close()

	for _, file := range reader.File {
		lowerName := strings.ToLower(file.Name)
		if strings.HasSuffix(lowerName, "moddesc.xml") || lowerName == "moddesc.xml" {
			rc, err := file.Open()
			if err != nil {
				return ModDescriptor{}, err
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return ModDescriptor{}, err
			}

			var desc modDesc
			if err := xml.Unmarshal(data, &desc); err != nil {
				return ModDescriptor{}, err
			}

			return ModDescriptor{
				Version: desc.Version,
				Author:  desc.Author,
			}, nil
		}
	}

	return ModDescriptor{}, nil
}

func ModExists(directory, filename string) bool {
	path := filepath.Join(directory, filename)
	_, err := os.Stat(path)
	return err == nil
}
