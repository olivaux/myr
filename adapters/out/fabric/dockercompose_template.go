// adapters/out/fabric/dockercompose_template.go
// Génère un docker-compose.yaml (CA + orderer Raft mono-nœud + N peers + CouchDB)
// pour le bootstrap d'un réseau from scratch, en réutilisant les images Fabric déjà
// présentes sur le serveur (hyperledger/fabric-ca:1.5.12, fabric-orderer:2.5.10,
// fabric-peer:2.5.10, couchdb:3.3).
package fabric

import (
	"bytes"
	"text/template"
)

type dcPeerParams struct {
	Name          string // ex: peer0.org1.diy-network.com
	Port          int    // gRPC (7051, 7061, ...)
	ChaincodePort int    // 7052, 7062, ...
	OperationPort int    // 9444, 9445, ...
	CouchDBName   string
	CouchDBPort   int
}

type dcOrdererParams struct {
	Name      string
	Port      int
	AdminPort int
	OpPort    int
}

type dockerComposeParams struct {
	NetworkName string // nom du réseau docker (bridge dédié)
	CryptoDir   string // répertoire hôte contenant le matériel crypto (monté en volume)

	CAName     string
	CAAdminID  string
	CAAdminPwd string
	CAPort     int

	OrgMSPID string

	// Orderers liste tous les orderers du réseau (1 par défaut — Raft mono-nœud).
	// Le premier élément est celui utilisé pour le bootstrap initial ; les suivants
	// sont ajoutés via `myr node add --type orderer`.
	Orderers []dcOrdererParams

	Peers []dcPeerParams
}

const dockerComposeTemplateText = `# docker-compose.yaml — généré par myr (network create)
version: "3.7"

networks:
  {{.NetworkName}}:
    name: {{.NetworkName}}

services:
  ca.{{.CAName}}:
    image: hyperledger/fabric-ca:1.5.12
    container_name: ca.{{.CAName}}
    environment:
      - FABRIC_CA_SERVER_HOME=/etc/hyperledger/fabric-ca-server
      - FABRIC_CA_SERVER_CA_NAME={{.CAName}}
      - FABRIC_CA_SERVER_TLS_ENABLED=true
      - FABRIC_CA_SERVER_PORT=7054
    ports:
      - "{{.CAPort}}:7054"
    command: sh -c 'fabric-ca-server start -b {{.CAAdminID}}:{{.CAAdminPwd}} -d'
    volumes:
      - {{.CryptoDir}}/ca:/etc/hyperledger/fabric-ca-server
    networks:
      - {{.NetworkName}}

{{range .Orderers}}
  {{.Name}}:
    image: hyperledger/fabric-orderer:2.5.10
    container_name: {{.Name}}
    environment:
      - FABRIC_LOGGING_SPEC=INFO
      - ORDERER_GENERAL_LISTENADDRESS=0.0.0.0
      - ORDERER_GENERAL_LISTENPORT=7050
      - ORDERER_GENERAL_BOOTSTRAPMETHOD=none
      - ORDERER_CHANNELPARTICIPATION_ENABLED=true
      - ORDERER_GENERAL_LOCALMSPID=OrdererMSP
      - ORDERER_GENERAL_LOCALMSPDIR=/var/hyperledger/orderer/msp
      - ORDERER_GENERAL_TLS_ENABLED=true
      - ORDERER_GENERAL_TLS_PRIVATEKEY=/var/hyperledger/orderer/tls/server.key
      - ORDERER_GENERAL_TLS_CERTIFICATE=/var/hyperledger/orderer/tls/server.crt
      - ORDERER_GENERAL_TLS_ROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]
      - ORDERER_ADMIN_TLS_ENABLED=true
      - ORDERER_ADMIN_TLS_CERTIFICATE=/var/hyperledger/orderer/tls/server.crt
      - ORDERER_ADMIN_TLS_PRIVATEKEY=/var/hyperledger/orderer/tls/server.key
      - ORDERER_ADMIN_TLS_CLIENTROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]
      - ORDERER_ADMIN_TLS_CLIENTAUTHREQUIRED=true
      - ORDERER_ADMIN_LISTENADDRESS=0.0.0.0:7053
      - ORDERER_OPERATIONS_LISTENADDRESS=0.0.0.0:{{.OpPort}}
    ports:
      - "{{.Port}}:7050"
      - "{{.AdminPort}}:7053"
    volumes:
      - {{$.CryptoDir}}/orderers/{{.Name}}/msp:/var/hyperledger/orderer/msp
      - {{$.CryptoDir}}/orderers/{{.Name}}/tls:/var/hyperledger/orderer/tls
      - {{.Name}}-data:/var/hyperledger/production/orderer
    networks:
      - {{$.NetworkName}}
{{end}}
{{range .Peers}}
  {{.CouchDBName}}:
    image: couchdb:3.3
    container_name: {{.CouchDBName}}
    environment:
      - COUCHDB_USER=admin
      - COUCHDB_PASSWORD=adminpw
    ports:
      - "{{.CouchDBPort}}:5984"
    networks:
      - {{$.NetworkName}}

  {{.Name}}:
    image: hyperledger/fabric-peer:2.5.10
    container_name: {{.Name}}
    environment:
      - CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock
      - CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE={{$.NetworkName}}
      - FABRIC_LOGGING_SPEC=INFO
      - CORE_PEER_TLS_ENABLED=true
      - CORE_PEER_TLS_CERT_FILE=/etc/hyperledger/fabric/tls/server.crt
      - CORE_PEER_TLS_KEY_FILE=/etc/hyperledger/fabric/tls/server.key
      - CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt
      - CORE_PEER_ID={{.Name}}
      - CORE_PEER_ADDRESS={{.Name}}:{{.Port}}
      - CORE_PEER_LISTENADDRESS=0.0.0.0:{{.Port}}
      - CORE_PEER_CHAINCODEADDRESS={{.Name}}:{{.ChaincodePort}}
      - CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:{{.ChaincodePort}}
      - CORE_PEER_GOSSIP_BOOTSTRAP={{.Name}}:{{.Port}}
      - CORE_PEER_GOSSIP_EXTERNALENDPOINT={{.Name}}:{{.Port}}
      - CORE_PEER_LOCALMSPID={{$.OrgMSPID}}
      - CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/msp
      - CORE_LEDGER_STATE_STATEDATABASE=CouchDB
      - CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS={{.CouchDBName}}:5984
      - CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME=admin
      - CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD=adminpw
      - CORE_OPERATIONS_LISTENADDRESS=0.0.0.0:{{.OperationPort}}
    ports:
      - "{{.Port}}:{{.Port}}"
    volumes:
      - /var/run/docker.sock:/host/var/run/docker.sock
      - {{$.CryptoDir}}/peers/{{.Name}}/msp:/etc/hyperledger/fabric/msp
      - {{$.CryptoDir}}/peers/{{.Name}}/tls:/etc/hyperledger/fabric/tls
      - {{.Name}}-data:/var/hyperledger/production
    depends_on:
      - {{.CouchDBName}}
    networks:
      - {{$.NetworkName}}
{{end}}
volumes:
{{range .Orderers}}  {{.Name}}-data:
{{end}}{{range .Peers}}  {{.Name}}-data:
{{end}}`

var dockerComposeTmpl = template.Must(template.New("docker-compose").Parse(dockerComposeTemplateText))

// renderDockerCompose produit le contenu du docker-compose.yaml pour les paramètres donnés.
func renderDockerCompose(p dockerComposeParams) (string, error) {
	var buf bytes.Buffer
	if err := dockerComposeTmpl.Execute(&buf, p); err != nil {
		return "", err
	}
	return buf.String(), nil
}
