package model

import "errors"

var ErrNotFound = errors.New("model: not found")
var ErrIntegrityFailed = errors.New("model: integrity check failed")

// ErrBlockchainUnavailable est retournée par toute méthode du service qui
// nécessite BlockchainPort quand celui-ci n'est pas configuré (aucun adapter
// blockchain injecté). Le service ne substitue jamais silencieusement un
// autre stockage — voir domain/channel.ErrFabricUnavailable pour le même
// principe appliqué au domaine channel.
var ErrBlockchainUnavailable = errors.New("model: adapter blockchain non configuré")

// ErrNoThumbnailSource est retournée par Service.RegenerateThumbnail quand l'asset
// n'a aucun lien externe (Links) — seule source de miniature régénérable côté serveur.
var ErrNoThumbnailSource = errors.New("model: aucun lien externe enregistré pour régénérer la miniature")
