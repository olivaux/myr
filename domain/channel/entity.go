// domain/channel/entity.go
package channel

// Channel représente un canal Hyperledger Fabric auquel l'utilisateur a accès.
// Les canaux sont créés et gérés par l'administrateur du réseau.
// L'application les liste en lecture seule depuis le profil de connexion.
type Channel struct {
	ID   string // nom du canal Fabric (ex: "greenchannel")
	Name string // nom d'affichage (= ID pour les canaux Fabric)
}

// Organization représente une organisation membre d'un canal Fabric.
// Le MSPID doit respecter le format [a-zA-Z0-9_.-]{1,128}.
type Organization struct {
	MSPID    string // ex: "Org1MSP"
	Name     string // nom lisible
	Role     string // "member" | "admin"
	RootCert string // certificat CA racine PEM
	TLSCert  string // certificat TLS CA racine PEM (optionnel)
}

// NodeType identifie le type de nœud Fabric.
type NodeType string

const (
	NodeTypePeer    NodeType = "peer"
	NodeTypeOrderer NodeType = "orderer"
)

// NodeCerts contient les certificats TLS d'un nœud Fabric.
type NodeCerts struct {
	TLSCert string // certificat TLS PEM (optionnel)
}
