// adapters/out/fabric/core_yaml.go
// Le CLI `peer` (exécuté directement sur l'hôte, hors conteneur, pour
// channel join/fetch/signconfigtx/update) exige un fichier core.yaml lisible
// via FABRIC_CFG_PATH, même pour ces sous-commandes qui ignorent la plupart
// de ses valeurs (écrasées par les variables CORE_PEER_* explicites). Sans
// lui, la commande échoue immédiatement : "Config File core Not Found".
// On embarque le core.yaml de référence (hyperledger/fabric/sampleconfig,
// Apache-2.0) et on l'écrit dans le répertoire de travail de chaque réseau.
package fabric

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed assets/core.yaml
var defaultCoreYAML []byte

// ensureCoreYAML écrit core.yaml dans dir s'il n'y est pas déjà, et retourne
// la variable d'environnement FABRIC_CFG_PATH à passer aux commandes `peer`.
func ensureCoreYAML(dir string) (string, error) {
	path := filepath.Join(dir, "core.yaml")
	if _, err := os.Stat(path); err == nil {
		return "FABRIC_CFG_PATH=" + dir, nil
	}
	if err := os.WriteFile(path, defaultCoreYAML, 0o644); err != nil {
		return "", err
	}
	return "FABRIC_CFG_PATH=" + dir, nil
}
