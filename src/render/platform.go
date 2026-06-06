package render

import (
	"runtime"

	"gioui.org/io/key"
)

// raccourcis selon la plateforme : sur macOS tout (edition, aide, presse-papier,
// QUITTER) passe par la touche Command la ou les autres OS prennent Ctrl. cmdMod
// sert au filtrage clavier et aux tests de modificateur ; modName / modAbbr a
// l'affichage dans les barres d'etat. basicfont est une police ASCII et ne connait
// pas le glyphe pomme, alors on ecrit "Cmd". decide une fois pour toutes au demarrage
var (
	cmdMod  = key.ModCtrl // modificateur des raccourcis : Command sur Mac, Ctrl ailleurs
	modName = "Ctrl"      // libelle long (barre d'etat editeur : "Ctrl+S")
	modAbbr = "C"         // forme courte (pied d'aide compact : "[C-i]")
)

func init() {
	if runtime.GOOS == "darwin" {
		cmdMod = key.ModCommand
		modName = "Cmd"
		modAbbr = "Cmd"
	}
}
