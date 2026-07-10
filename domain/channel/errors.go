package channel

import "errors"

var (
	ErrNotFound          = errors.New("channel: not found")
	ErrAlreadyExists     = errors.New("channel: already exists")
	ErrAlreadyMember     = errors.New("channel: organisation already member")
	ErrInvalidMSPID      = errors.New("channel: MSP ID invalide — caractères non autorisés ou longueur hors limites (max 128)")
	ErrFabricUnavailable = errors.New("channel: adapter Fabric non configuré")
	ErrEndorsementPolicy = errors.New("channel: politique d'endorsement non satisfaite")
	ErrNodeUnreachable   = errors.New("channel: nœud inaccessible")
	ErrSyncTimeout       = errors.New("channel: synchronisation du ledger hors délai")
	ErrMinNodesRequired  = errors.New("channel: le réseau doit conserver au moins 3 nœuds actifs [RM27]")
	ErrNodeNotMember     = errors.New("channel: nœud non membre du canal")
)
