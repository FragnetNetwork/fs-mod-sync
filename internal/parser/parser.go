package parser

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/FragnetNetwork/fs-mod-sync/internal/models"

	"github.com/PuerkitoBio/goquery"
)

func FetchAndParse(serverURL string) ([]models.Mod, string, error) {
	resp, err := http.Get(serverURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	return ParseHTML(string(body), serverURL)
}

func ParseHTML(html string, baseURL string) ([]models.Mod, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	gameVersion := detectGameVersion(html)
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid base URL: %w", err)
	}

	var mods []models.Mod

	doc.Find(".container-row.grid-row").Each(func(i int, row *goquery.Selection) {
		nameDiv := row.Find(`div[title]`).First()
		title, exists := nameDiv.Attr("title")
		if !exists || title == "" {
			return
		}

		if strings.Contains(title, "Total") && strings.Contains(title, "Mods") {
			return
		}

		mod := models.Mod{
			Name: title,
		}

		row.Find(".container-row").Each(func(j int, col *goquery.Selection) {
			colTitle, _ := col.Attr("title")

			if strings.Contains(col.Text(), "Version") || isVersionFormat(colTitle) {
				if colTitle != "" && isVersionFormat(colTitle) {
					mod.Version = colTitle
				}
			}

			if strings.Contains(col.Text(), "Author") {
				if colTitle != "" {
					mod.Author = colTitle
				}
			}

			if strings.Contains(col.Text(), "Filename") || strings.HasSuffix(colTitle, ".zip") || strings.HasSuffix(colTitle, ".dlc") {
				if colTitle != "" {
					mod.Filename = colTitle
					mod.IsDLC = strings.HasSuffix(colTitle, ".dlc")
				}
			}

			if strings.Contains(col.Text(), "Size") {
				if colTitle != "" && (strings.Contains(colTitle, "MB") || strings.Contains(colTitle, "GB") || strings.Contains(colTitle, "KB")) {
					mod.Size = colTitle
					mod.SizeBytes = parseSize(colTitle)
				}
			}

			if strings.Contains(col.Text(), "Active") {
				mod.IsActive = strings.Contains(col.Text(), "Yes")
			}
		})

		row.Find("a[href]").Each(func(k int, link *goquery.Selection) {
			href, exists := link.Attr("href")
			if exists && strings.HasPrefix(href, "mods/") && strings.HasSuffix(href, ".zip") {
				modURL, err := url.Parse(href)
				if err == nil {
					mod.URL = base.ResolveReference(modURL).String()
				}
			}
		})

		if mod.Filename == "" {
			filenameLink := row.Find(`a[href^="mods/"]`)
			if filenameLink.Length() > 0 {
				href, _ := filenameLink.Attr("href")
				mod.Filename = strings.TrimPrefix(href, "mods/")
				mod.IsDLC = strings.HasSuffix(mod.Filename, ".dlc")
			}
		}

		if mod.Filename != "" {
			mods = append(mods, mod)
		}
	})

	return mods, gameVersion, nil
}

func detectGameVersion(html string) string {
	if strings.Contains(html, "10.0.0.0") {
		return "FS25"
	}
	return "FS22"
}

func isVersionFormat(s string) bool {
	matched, _ := regexp.MatchString(`^\d+\.\d+\.\d+\.\d+$`, s)
	return matched
}

func parseSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(sizeStr)
	re := regexp.MustCompile(`([\d.]+)\s*(KB|MB|GB)`)
	matches := re.FindStringSubmatch(sizeStr)
	if len(matches) < 3 {
		return 0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToUpper(matches[2])
	switch unit {
	case "KB":
		return int64(value * 1024)
	case "MB":
		return int64(value * 1024 * 1024)
	case "GB":
		return int64(value * 1024 * 1024 * 1024)
	}

	return 0
}

func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func IsValidModsPage(html string) bool {
	return strings.Contains(html, `href="mods/`) && strings.Contains(html, `.zip"`)
}
