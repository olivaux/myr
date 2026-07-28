// cmd/api/main.go — point d'entrée principal de myr-core.
//
// Lance un serveur HTTP local qui sert exclusivement l'API REST — aucune
// interface graphique n'est embarquée. Le GUI est un dépôt externe, client
// de cette API.
//
// Usage minimal (desktop local) :
//
//	myr-api                          # lit ~/.Myr/data/fabric.env automatiquement
//	myr-api --addr :9000             # port personnalisé
//	myr-api --env-file /chemin/.env  # fichier de config explicite
//
// Usage serveur :
//
//	myr-api --addr 0.0.0.0:8080 --data /var/myr
//
//	@title			Myr API
//	@version		1.0
//	@description	API REST de myr-core — blockchain, stockage fichiers, identités, sessions. Le CLI (`myr-cli`) et tout logiciel tiers (GUI, plugin CAO, script) consomment cette même API via le domaine (`domain/`).
//	@BasePath		/api
//	@schemes		http https
//
//	@securityDefinitions.apikey	MyrToken
//	@in							header
//	@name						X-Myr-Token
//	@description				Token opaque de session (32 octets hex), obtenu via POST /identity/session, /identity/guest ou /identity/enroll — pas de JWT (voir la règle d'authentification du domaine identité).
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"myr-core/adapters/in/rest"
	fabricadapter "myr-core/adapters/out/fabric"
	"myr-core/adapters/out/localstorage"
	"myr-core/adapters/out/webimage"
	"myr-core/domain/identity"
	"myr-core/domain/model"
	"myr-core/domain/network"
	"myr-core/domain/role"
)

// BuildDate est injecté via -ldflags "-X main.BuildDate=..." lors de la compilation.
var BuildDate = "dev"

// version est injectée via -ldflags "-X main.version=..." lors de la compilation
// (voir Makefile : VERSION dérivé de `git describe --tags`).
var version = "dev"

func main() {
	addr        := flag.String("addr", "localhost:8765", "adresse d'écoute (ex: localhost:8765 ou 0.0.0.0:8080)")
	dataDir     := flag.String("data", defaultDataDir(), "répertoire de données")
	envFile     := flag.String("env-file", "", "fichier .env explicite (sinon auto-détecté)")
	showVersion := flag.Bool("version", false, "affiche la version et quitte")
	flag.Parse()

	if *showVersion {
		fmt.Printf("myr-api %s (build %s)\n", version, BuildDate)
		return
	}

	// Auto-détection du fichier fabric.env si non fourni explicitement.
	if *envFile == "" {
		candidate := filepath.Join(userDataDir(), "fabric.env")
		if _, err := os.Stat(candidate); err == nil {
			*envFile = candidate
		}
	}
	if *envFile != "" {
		if err := loadEnvFile(*envFile); err != nil {
			log.Fatalf("impossible de lire %s : %v", *envFile, err)
		}
		log.Printf("config chargée depuis %s", *envFile)
	}

	cfg := fabricadapter.ConfigFromEnv()

	// ── Connexion Gateway Fabric (non bloquante) ──────────────────────────────
	// bc reste nil si Fabric est indisponible — pas de repli silencieux vers un
	// autre stockage : les opérations sur les modèles retournent alors
	// model.ErrBlockchainUnavailable (voir domain/model/service.go).
	var bc model.BlockchainPort
	var fabricConnected bool
	gw, err := fabricadapter.NewGatewayClient(cfg)
	if err != nil {
		log.Printf("⚠  Fabric non disponible : %v", err)
		log.Printf("   Les opérations sur les modèles échoueront tant que Fabric n'est pas configuré.")
	} else {
		defer gw.Close()
		log.Printf("✓ Fabric → %s (canal: %s)", cfg.PeerEndpoint, cfg.FabricChannel)
		bc = fabricadapter.NewFabricBlockchain(gw)
		fabricConnected = true
	}

	// ── Connexion CA (optionnelle) ────────────────────────────────────────────
	var caPort identity.CAPort
	if cfg.CAEndpoint != "" {
		if ca, err := fabricadapter.NewCAClient(cfg); err != nil {
			log.Printf("⚠  CA non disponible : %v", err)
		} else if ca != nil {
			log.Printf("✓ CA Fabric → %s (%s)", cfg.CAEndpoint, cfg.CAName)
			caPort = ca
		}
	}

	// ── Stockage fichiers ─────────────────────────────────────────────────────
	store := localstorage.NewJSONBlockchain(filepath.Join(*dataDir, "assets.json"))

	modelsDir := filepath.Join(*dataDir, "models")
	_ = os.MkdirAll(modelsDir, 0700)
	var fileStore model.FileStoragePort = localstorage.NewLocalStorage(modelsDir)
	log.Printf("✓ Stockage local → %s", modelsDir)

	// ── Services domaine ──────────────────────────────────────────────────────
	modelSvc := model.NewService(bc, fileStore).
		WithConnStore(store).
		WithThumbStore(store).
		WithIfaceStore(store).
		WithDraftStore(store).
		WithOGImageFetcher(webimage.New())

	walletDir := filepath.Join(*dataDir, "wallets")
	_ = os.MkdirAll(walletDir, 0700)
	requestStore := localstorage.NewJSONRequestStore(filepath.Join(*dataDir, "account-requests.json"))
	identitySvc := identity.NewService(walletDir, caPort).WithRequestStore(requestStore)

	roleStore := localstorage.NewJSONRoleStore(filepath.Join(*dataDir, "roles.json"))
	roleSvc := role.NewService(roleStore)

	// ── Service réseau + NetworkPool multi-réseau ────────────────────────────
	networkStore := localstorage.NewJSONNetworkStore(filepath.Join(*dataDir, "networks.json"))
	networkSvc := network.NewService(networkStore, nil)

	pool := fabricadapter.NewNetworkPool()
	var defaultNetworkID string
	if profiles, err := networkSvc.List(); err == nil && len(profiles) > 0 {
		pool.LoadFromProfiles(profiles)
		// Réseau par défaut = premier profil actif, sinon le premier de la liste
		for _, p := range profiles {
			if p.Active {
				defaultNetworkID = p.ID
				break
			}
		}
		if defaultNetworkID == "" {
			defaultNetworkID = profiles[0].ID
		}
	}
	defer pool.Close()

	// ── Handler REST + serving de l'UI ───────────────────────────────────────
	netInfo := rest.NetworkInfo{
		Channel:  cfg.FabricChannel,
		Channels: []string{cfg.FabricChannel},
		Peer:     cfg.GatewayPeer,
		Network:  cfg.PeerEndpoint,
	}
	if cfg.FabricChannel == "" {
		netInfo.Channel = "sandbox"
		netInfo.Channels = []string{"sandbox"}
	}

	handler := rest.NewHandler(modelSvc, store, store).
		WithIdentityService(identitySvc).
		WithNetworkService(networkSvc).
		WithRoleService(roleSvc).
		WithNetworkInfo(netInfo).
		WithSessionPersistence(filepath.Join(*dataDir, "sessions.json")).
		WithBlockchainRouter(pool).
		WithFileStore(fileStore).
		WithConnStore(store).
		WithDraftStore(store).
		WithDefaultNetwork(defaultNetworkID).
		WithFabricConnected(fabricConnected).
		WithBuildDate(BuildDate).
		WithVersion(version)

	// Redis optionnel — si REDIS_URL est défini, les sessions sont partagées entre instances.
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		handler = handler.WithRedisSession(redisURL)
		log.Printf("✓ Redis sessions → %s", redisURL)
	}

	server := rest.NewServer(handler, *addr)

	url := "http://" + *addr
	fmt.Printf("\n  Myr  →  %s\n\n", url)
	if cfg.PeerEndpoint != "" {
		fmt.Printf("  Fabric  : %s  (canal: %s)\n", cfg.PeerEndpoint, cfg.FabricChannel)
	} else {
		fmt.Printf("  Mode    : blockchain indisponible — fonctionnalités liées aux modèles désactivées tant que fabric.env n'est pas configuré\n")
	}
	fmt.Printf("  Données : %s\n\n", *dataDir)
	fmt.Println("  Arrêt : Ctrl+C")
	fmt.Println()

	// Graceful shutdown : SIGTERM / SIGINT → arrêt propre en 15 secondes max.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := server.StartWithShutdown(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
	log.Println("Arrêt terminé.")
}

// loadEnvFile lit un fichier KEY=VALUE et injecte les valeurs dans l'environnement.
func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
	return scanner.Err()
}

// userDataDir retourne ~/.Myr/data (données utilisateur desktop).
func userDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".Myr", "data")
}

// defaultDataDir retourne le répertoire de données par défaut.
func defaultDataDir() string {
	dir := filepath.Join(userDataDir(), "..", "server-data")
	dir = filepath.Clean(dir)
	_ = os.MkdirAll(dir, 0755)
	return dir
}
