// Package ipfs implémente model.FileStoragePort via le daemon IPFS local (kubo).
// Upload retourne le CID du fichier ajouté ; ce CID est stocké sur Fabric comme hash.
// Le CID est lui-même le hash cryptographique du contenu (SHA-256 en CIDv1),
// ce qui garantit l'intégrité sans vérification externe.
package ipfs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// IPFSStorage implémente model.FileStoragePort en utilisant l'API HTTP du daemon kubo.
type IPFSStorage struct {
	cfg    Config
	client *http.Client
}

// New crée un IPFSStorage prêt à l'emploi.
func New(cfg Config) *IPFSStorage {
	return &IPFSStorage{
		cfg:    cfg,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

type addResponse struct {
	Hash string `json:"Hash"`
}

// Upload envoie le fichier au daemon IPFS et retourne son CID (= hash du contenu).
// Le fichier est automatiquement pinné pour éviter le garbage collection.
func (s *IPFSStorage) Upload(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("ipfs: open %s: %w", filePath, err)
	}
	defer f.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("ipfs: create form: %w", err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		return "", fmt.Errorf("ipfs: read file: %w", err)
	}
	mw.Close()

	url := s.cfg.APIEndpoint + "/api/v0/add?pin=true&cid-version=1"
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return "", fmt.Errorf("ipfs: build request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ipfs: add request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ipfs: daemon %d: %s", resp.StatusCode, b)
	}

	var result addResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("ipfs: decode response: %w", err)
	}
	return result.Hash, nil
}

// Download récupère le contenu identifié par son CID et l'écrit dans destPath.
func (s *IPFSStorage) Download(cid, destPath string) error {
	url := s.cfg.APIEndpoint + "/api/v0/cat?arg=" + cid
	resp, err := s.client.Post(url, "", nil)
	if err != nil {
		return fmt.Errorf("ipfs: cat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ipfs: daemon %d: %s", resp.StatusCode, b)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("ipfs: create %s: %w", destPath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Delete supprime le pin local du CID. Le bloc sera collecté au prochain GC.
// Le contenu reste accessible sur le réseau IPFS tant qu'un autre nœud le pine.
func (s *IPFSStorage) Delete(cid string) error {
	url := s.cfg.APIEndpoint + "/api/v0/pin/rm?arg=" + cid
	resp, err := s.client.Post(url, "", nil)
	if err != nil {
		return fmt.Errorf("ipfs: pin/rm request: %w", err)
	}
	defer resp.Body.Close()

	// 500 avec "not pinned" est acceptable (idempotent)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ipfs: daemon %d: %s", resp.StatusCode, b)
	}
	return nil
}
