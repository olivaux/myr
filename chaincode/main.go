// chaincode/main.go
package main

import (
	"fmt"
	"os"

	"github.com/hyperledger/fabric-chaincode-go/shim"

	"myrcc/model"
)

// main démarre myrcc en mode chaincode-as-a-service (CCaaS) : le peer s'y
// connecte via l'adresse déclarée dans connection.json au packaging, au lieu
// de construire/lancer lui-même un conteneur (voir chaincode/Dockerfile).
func main() {
	address := os.Getenv("CHAINCODE_SERVER_ADDRESS")
	if address == "" {
		address = "0.0.0.0:9999"
	}
	server := &shim.ChaincodeServer{
		CCID:     os.Getenv("CHAINCODE_ID"),
		Address:  address,
		CC:       new(model.SmartContract),
		TLSProps: shim.TLSProperties{Disabled: true},
	}
	if err := server.Start(); err != nil {
		fmt.Printf("erreur démarrage chaincode server myrcc : %v\n", err)
		os.Exit(1)
	}
}
