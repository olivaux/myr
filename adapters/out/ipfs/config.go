package ipfs

import "os"

// Config contient les paramètres de connexion au daemon IPFS local.
type Config struct {
	// Endpoint de l'API RPC locale — ne jamais exposer publiquement.
	APIEndpoint string
	// URL de la gateway HTTP pour téléchargement public (optionnel).
	GatewayURL string
}

// ConfigFromEnv charge la configuration depuis les variables d'environnement.
func ConfigFromEnv() Config {
	return Config{
		APIEndpoint: getEnv("IPFS_API_ENDPOINT", "http://127.0.0.1:5001"),
		GatewayURL:  getEnv("IPFS_GATEWAY_URL", "http://127.0.0.1:8080"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
