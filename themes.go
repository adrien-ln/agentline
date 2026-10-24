// Package agentline expose les thèmes intégrés au binaire (les mêmes fichiers que la version bash).
package agentline

import "embed"

// Themes contient themes/*.theme, lus par le binaire sans accès disque.
//
//go:embed themes/*.theme
var Themes embed.FS
