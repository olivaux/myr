// adapters/out/webimage/fetcher.go — adapter de récupération d'image représentative
// (balise og:image) depuis une page web, pour le port model.OGImageFetcher.
package webimage

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var reOGImage = regexp.MustCompile(
	`(?i)<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']+)["']|` +
		`<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:image["']`,
)

// Fetcher implémente model.OGImageFetcher via des requêtes HTTP directes.
type Fetcher struct{}

func New() *Fetcher { return &Fetcher{} }

// FetchOGImage récupère l'image og:image d'une page web et la retourne en data URL base64.
func (Fetcher) FetchOGImage(pageURL string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(pageURL)
	if err != nil {
		return "", fmt.Errorf("chargement page: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2 Mo max
	if err != nil {
		return "", err
	}

	matches := reOGImage.FindSubmatch(body)
	imgURL := ""
	if len(matches) >= 3 {
		if len(matches[1]) > 0 {
			imgURL = string(matches[1])
		} else {
			imgURL = string(matches[2])
		}
	}
	if imgURL == "" {
		return "", fmt.Errorf("og:image introuvable sur %s", pageURL)
	}

	// Résoudre les URLs relatives
	if !strings.HasPrefix(imgURL, "http") {
		base, err := url.Parse(pageURL)
		if err != nil {
			return "", err
		}
		ref, err := url.Parse(imgURL)
		if err != nil {
			return "", err
		}
		imgURL = base.ResolveReference(ref).String()
	}

	imgResp, err := client.Get(imgURL)
	if err != nil {
		return "", fmt.Errorf("chargement image: %w", err)
	}
	defer imgResp.Body.Close()
	imgData, err := io.ReadAll(io.LimitReader(imgResp.Body, 5<<20)) // 5 Mo max
	if err != nil {
		return "", err
	}

	ct := strings.Split(imgResp.Header.Get("Content-Type"), ";")[0]
	if ct == "" {
		ct = "image/jpeg"
	}
	return fmt.Sprintf("data:%s;base64,%s", ct, base64.StdEncoding.EncodeToString(imgData)), nil
}
