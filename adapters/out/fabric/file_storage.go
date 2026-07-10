// adapters/out/fabric/file_storage.go
package fabric

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ContentAddressedStorage implémente model.FileStoragePort avec un stockage
// local adressé par contenu (SHA-256), compatible avec l'ancrage Fabric.
// Les fichiers sont nommés par leur hash — identique au principe de Git objects.
type ContentAddressedStorage struct {
	basePath string
}

func NewContentAddressedStorage(basePath string) (*ContentAddressedStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("content storage mkdir : %w", err)
	}
	return &ContentAddressedStorage{basePath: basePath}, nil
}

// Upload copie le fichier dans le store et retourne son SHA-256 hex.
func (s *ContentAddressedStorage) Upload(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("upload open : %w", err)
	}
	defer f.Close()

	// Calcul du hash SHA-256
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("upload hash : %w", err)
	}
	hash := hex.EncodeToString(h.Sum(nil))

	// Copie vers basePath/<hash>
	dest := filepath.Join(s.basePath, hash)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		// Rembobinage pour relire le fichier
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return "", fmt.Errorf("upload seek : %w", err)
		}
		dst, err := os.Create(dest)
		if err != nil {
			return "", fmt.Errorf("upload create dest : %w", err)
		}
		defer dst.Close()
		if _, err := io.Copy(dst, f); err != nil {
			return "", fmt.Errorf("upload copy : %w", err)
		}
	}

	return hash, nil
}

// Download copie le fichier identifié par son hash vers destPath.
func (s *ContentAddressedStorage) Download(hash, destPath string) error {
	src, err := os.Open(filepath.Join(s.basePath, hash))
	if err != nil {
		return fmt.Errorf("download open : %w", err)
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("download create dest : %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("download copy : %w", err)
	}
	return nil
}

// Delete supprime le fichier du store.
func (s *ContentAddressedStorage) Delete(hash string) error {
	err := os.Remove(filepath.Join(s.basePath, hash))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
