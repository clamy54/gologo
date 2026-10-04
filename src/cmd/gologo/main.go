// commande gologo : l'interpreteur Logo dans une fenetre Gio.
// gologo seul = plein ecran (Ctrl+Q pour sortir), -w pour une fenetre (dev),
// -x pour forcer le dossier des exemples, -e pour demarrer en anglais
package main

import (
	"flag"
	"log"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/unit"

	"beroot.com/logo/audio"
	"beroot.com/logo/logo"
	"beroot.com/logo/render"
	"beroot.com/logo/turtle"
)

func main() {
	fenetre := flag.Bool("w", false, "lance en fenêtré (windowed) au lieu du plein écran")
	exec := flag.String("exec", "", "ligne Logo exécutée au démarrage (démo/cours)")
	exDir := flag.String("x", "", "dossier des exemples (force son emplacement)")
	anglais := flag.Bool("e", false, "démarre en anglais (English) ; par défaut en français")
	flag.Parse()

	// les 4 briques, puis on les relie
	screen := render.New()
	tort := turtle.New(screen)       // la tortue dessine sur l'ecran
	interp := logo.New(tort, screen) // l'interp ecrit le texte

	// callbacks dans les deux sens entre l'ecran et l'interpreteur
	screen.SetRunner(interp.RunString)      // execute une ligne tapee
	screen.SetInterrupt(interp.Interrupt)   // Ctrl+C = break
	screen.SetCompleter(interp.Completions) // completion au TAB
	interp.SetEditor(screen.Edit)           // ED : editeur plein ecran
	interp.SetKeyboard(screen)              // LISCAR/LL/TOUCHE?
	interp.SetMouse(screen)                 // POSOPT/CONTACT?
	interp.SetJoystick(screen)              // manettes emulees au clavier
	son := audio.New()
	son.SetCancel(interp.Interrupted)       // Ctrl+C coupe la note en cours
	interp.SetSound(son)                    // JOUE
	interp.SetHelper(screen.Help)           // AIDE
	screen.SetHelpOpener(interp.HelpOpen)   // F1
	screen.SetHelpResolver(interp.HelpName) // F1 sur un mot : sa fiche
	screen.SetLangFunc(interp.Lang)
	screen.SetTranslator(interp.TranslateProgram) // Ctrl+T traduit FR/EN
	screen.SetErrorText(interp.ErrorText)
	interp.SetPager(screen.Page) // sortie longue facon "more"
	screen.SetStartup(*exec)
	if *exDir != "" {
		interp.SetExamplesDir(*exDir) // -x force le dossier des exemples
	}
	if *anglais {
		interp.SetLang("EN") // -e : demarre en anglais (defaut francais)
	}

	// moteur d'anim (ANIME) : le goroutine dort sur wake et ne tourne que
	// pendant une animation, donc cpu quasi nul au repos
	wake := make(chan struct{}, 1)
	tort.SetAnimWake(func() {
		select {
		case wake <- struct{}{}:
		default: // un reveil deja en attente suffit
		}
	})
	go func() {
		for range wake {
			for tort.AnimStep() {
				fps := tort.FPS()
				if fps < 1 {
					fps = turtle.AnimDefaultFPS
				}
				time.Sleep(time.Second / time.Duration(fps))
			}
		}
	}()

	go func() {
		w := new(app.Window)
		w.Option(app.Title("GoLogo v" + logo.Version))
		if *fenetre {
			// demarrer maximise : evite le clignotement de la barre de titre sous KWin
			w.Option(app.Size(unit.Dp(render.ScreenW), unit.Dp(render.ScreenH)), app.Maximized.Option())
		} else {
			w.Option(app.Fullscreen.Option())
		}
		err := screen.Run(w)
		// la fenetre est fermee, mais un programme Logo tourne peut-etre encore :
		// on attend que sa tache s'arrete avant de refermer les fichiers de
		// donnees, qui lui appartiennent. si elle ne rend pas la main (appel
		// systeme bloquant), on n'y touche pas : le systeme les refermera a la
		// sortie, et les ecritures ne sont pas mises en tampon
		if screen.WaitWorker(2 * time.Second) {
			if cerr := interp.CloseFiles(); cerr != nil {
				log.Println("fermeture fichiers:", cerr)
			}
		}
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0) // la fenetre est morte, la tortue avec : RIP
	}()
	app.Main()
}
