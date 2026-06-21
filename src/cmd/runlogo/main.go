// commande runlogo : execute un fichier source Logo en mode texte, sans fenetre.
// pratique pour les tests et l'automatisation (comparaison avec d'autres langages).
// usage : runlogo [-d dossier] fichier.logo
package main

import (
	"flag"
	"fmt"
	"os"

	"beroot.com/logo/logo"
	"beroot.com/logo/turtle"
)

func main() {
	dir := flag.String("d", ".", "dossier de travail (pour les fichiers de donnees)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: runlogo [-d dossier] fichier.logo")
		os.Exit(2)
	}
	src, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	in := logo.New(turtle.New(turtle.NewRecorder()), os.Stdout)
	in.Quiet = true
	in.SetWorkDir(*dir)
	if err := in.RunString(string(src)); err != nil {
		fmt.Fprintln(os.Stderr, "ERREUR:", in.ErrorText(err))
		if cerr := in.CloseFiles(); cerr != nil {
			fmt.Fprintln(os.Stderr, "fermeture fichiers:", cerr)
		}
		os.Exit(1)
	}
	if cerr := in.CloseFiles(); cerr != nil {
		fmt.Fprintln(os.Stderr, "fermeture fichiers:", cerr)
		os.Exit(1)
	}
}
