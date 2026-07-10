// adapters/out/fabric/configtx_template.go
// Génère un configtx.yaml minimal (1 org applicative + 1 orderer Raft mono-nœud)
// pour le bootstrap d'un réseau from scratch. Reprend la structure déjà utilisée
// manuellement dans scripts/create-sandbox-channel.sh.
package fabric

import (
	"bytes"
	"text/template"
)

// configtxParams paramètre la génération du configtx.yaml.
type configtxParams struct {
	OrgMSPID       string
	OrgName        string
	OrgMSPDir      string
	AnchorPeerHost string
	AnchorPeerPort int

	OrdererMSPID   string
	OrdererMSPDir  string
	OrdererHost    string
	OrdererPort    int
	OrdererTLSCert string

	ProfileName string
}

const configtxTemplateText = `# configtx.yaml — généré par myr (network create)
Organizations:
  - &Org
    # Name est utilisé par configtxgen comme clé du groupe de configuration du
    # canal (pas un simple libellé) : il doit être un identifiant valide, sans
    # espace ni caractère spécial — d'où l'usage du MSPID plutôt que du nom
    # lisible fourni par l'admin (--org-name), qui reste une donnée myr, jamais
    # transmise à Fabric.
    Name: {{.OrgMSPID}}
    ID: {{.OrgMSPID}}
    MSPDir: {{.OrgMSPDir}}
    Policies:
      Readers:
        Type: Signature
        Rule: "OR('{{.OrgMSPID}}.admin', '{{.OrgMSPID}}.peer', '{{.OrgMSPID}}.client')"
      Writers:
        Type: Signature
        Rule: "OR('{{.OrgMSPID}}.admin', '{{.OrgMSPID}}.client')"
      Admins:
        Type: Signature
        Rule: "OR('{{.OrgMSPID}}.admin')"
      Endorsement:
        Type: Signature
        Rule: "OR('{{.OrgMSPID}}.peer')"
    AnchorPeers:
      - Host: {{.AnchorPeerHost}}
        Port: {{.AnchorPeerPort}}

  - &OrdererOrg
    Name: {{.OrdererMSPID}}
    ID: {{.OrdererMSPID}}
    MSPDir: {{.OrdererMSPDir}}
    Policies:
      Readers:
        Type: Signature
        Rule: "OR('{{.OrdererMSPID}}.member')"
      Writers:
        Type: Signature
        Rule: "OR('{{.OrdererMSPID}}.member')"
      Admins:
        Type: Signature
        Rule: "OR('{{.OrdererMSPID}}.admin')"

Capabilities:
  Channel: &ChannelCapabilities
    V2_0: true
  Orderer: &OrdererCapabilities
    V2_0: true
  Application: &ApplicationCapabilities
    V2_5: true

Application: &ApplicationDefaults
  Organizations:
  Policies:
    Readers:
      Type: ImplicitMeta
      Rule: "ANY Readers"
    Writers:
      Type: ImplicitMeta
      Rule: "ANY Writers"
    Admins:
      Type: ImplicitMeta
      Rule: "MAJORITY Admins"
    LifecycleEndorsement:
      Type: ImplicitMeta
      Rule: "MAJORITY Endorsement"
    Endorsement:
      Type: ImplicitMeta
      Rule: "MAJORITY Endorsement"
  Capabilities:
    <<: *ApplicationCapabilities

Orderer: &OrdererDefaults
  OrdererType: etcdraft
  Addresses:
    - {{.OrdererHost}}:{{.OrdererPort}}
  EtcdRaft:
    Consenters:
      - Host: {{.OrdererHost}}
        Port: {{.OrdererPort}}
        ClientTLSCert: {{.OrdererTLSCert}}
        ServerTLSCert: {{.OrdererTLSCert}}
  BatchTimeout: 2s
  BatchSize:
    MaxMessageCount: 10
    AbsoluteMaxBytes: 99 MB
    PreferredMaxBytes: 512 KB
  Policies:
    Readers:
      Type: ImplicitMeta
      Rule: "ANY Readers"
    Writers:
      Type: ImplicitMeta
      Rule: "ANY Writers"
    Admins:
      Type: ImplicitMeta
      Rule: "MAJORITY Admins"
    BlockValidation:
      Type: ImplicitMeta
      Rule: "ANY Writers"
  Capabilities:
    <<: *OrdererCapabilities

Channel: &ChannelDefaults
  Policies:
    Readers:
      Type: ImplicitMeta
      Rule: "ANY Readers"
    Writers:
      Type: ImplicitMeta
      Rule: "ANY Writers"
    Admins:
      Type: ImplicitMeta
      Rule: "MAJORITY Admins"
  Capabilities:
    <<: *ChannelCapabilities

Profiles:
  {{.ProfileName}}:
    <<: *ChannelDefaults
    Orderer:
      <<: *OrdererDefaults
      Organizations:
        - *OrdererOrg
      Capabilities: *OrdererCapabilities
    Application:
      <<: *ApplicationDefaults
      Organizations:
        - *Org
      Capabilities: *ApplicationCapabilities
`

var configtxTmpl = template.Must(template.New("configtx").Parse(configtxTemplateText))

// renderConfigtx produit le contenu du configtx.yaml pour les paramètres donnés.
func renderConfigtx(p configtxParams) (string, error) {
	if p.ProfileName == "" {
		p.ProfileName = "MyrNetworkGenesis"
	}
	var buf bytes.Buffer
	if err := configtxTmpl.Execute(&buf, p); err != nil {
		return "", err
	}
	return buf.String(), nil
}
