package logo

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"beroot.com/logo/turtle"
)

// tests de non-regression : chaque cas reprend un defaut corrige (audits statiques
// d'octobre 2026). tout tourne sans ecran, dans un dossier de travail jetable

type session struct {
	t   *testing.T
	in  *Interp
	out *bytes.Buffer
}

func newSession(t *testing.T) *session {
	t.Helper()
	out := &bytes.Buffer{}
	in := New(turtle.New(turtle.NewRecorder()), out)
	in.Quiet = true
	in.SetWorkDir(t.TempDir())
	return &session{t, in, out}
}

// execute src ; rend ce qui a ete affiche, suivi de "ERR:message" en cas d'erreur
func (s *session) run(src string) string {
	s.out.Reset()
	err := s.in.RunString(src)
	got := s.out.String()
	if err != nil {
		got += "ERR:" + err.Error()
	}
	return got
}

func (s *session) want(src, want string) {
	s.t.Helper()
	if got := s.run(src); got != want {
		s.t.Errorf("%s\n  obtenu  %q\n  attendu %q", src, got, want)
	}
}

func (s *session) ok(src string) {
	s.t.Helper()
	if got := s.run(src); got != "" {
		s.t.Fatalf("%s\n  %s", src, got)
	}
}

// --- lecteur ---

func TestLecteur(t *testing.T) {
	s := newSession(t)
	s.want("ECRIS 1 \x00 ECRIS 2", "ERR:CARACTERE NUL DANS LE PROGRAMME")
	s.want(`MONTRE {-5} MONTRE {1 2}@-2`, "{-5}\n{1 2}@-2\n")
	s.want(`ECRIS 1`+strings.Repeat("0", 320)+` > 1`, "VRAI\n")
	s.want(`ECRIS INF`, "ERR:COMMENT FAIRE INF")
	s.want(`ECRIS NOMBRE? "NAN`, "FAUX\n")
	s.want(`ECRIS `+strings.Repeat("MOINS ", 100000)+`1`, "ERR:PLUS DE PLACE")
}

// --- evaluateur ---

func TestPorteeDynamique(t *testing.T) {
	s := newSession(t)
	// un appel terminal ne fait pas disparaitre les variables de l'appelant
	s.want(`POUR B ECRIS :X FIN POUR A :X B FIN A 42`, "42\n")
	s.want(`POUR C :Y LOCAL "Z DONNE "Z 7 D FIN
POUR D ECRIS (LISTE :Y :Z) DONNE "Z 8 E FIN
POUR E ECRIS :Z FIN
C 1`, "1 7\n8\n")
	// ... et la recursion terminale tourne toujours a pile constante
	s.want(`POUR BOUCLE :N SI :N = 0 [ECRIS "FINI STOP] BOUCLE :N - 1 FIN BOUCLE 100000`, "FINI\n")
	s.want(`POUR PING :N :ACC SI :N = 0 [RENDS :ACC] RENDS PONG :N - 1 FIN
POUR PONG :M RENDS PING :M :ACC + 1 FIN
ECRIS PING 50000 0`, "50000\n")
}

func TestControle(t *testing.T) {
	s := newSession(t)
	s.want(`POUR P LANCE "SORTIE FIN POUR Q P FIN PIEGE "SORTIE [Q] ECRIS "OK`, "OK\n")
	s.want(`PIEGE "ERREUR [P]`, "ERR:PIEGE MANQUANT POUR SORTIE")
	s.want(`PIEGE "ERREUR [ECRIS 1 / 0] ECRIS "SUITE`, "SUITE\n")
	// RAZ dans une boucle ou une procedure : arret propre, pas d'erreur interne
	s.want(`POUR R REPETE 2 [RAZ] ECRIS "SUITE FIN R ECRIS "JAMAIS`, "")
	s.want(`ECRIS PROC? "R`, "FAUX\n")
	// formes entre parentheses : nombre d'arguments verifie
	s.want(`ECRIS (TABLEAU)`, "ERR:PAS ASSEZ DE DONNEES POUR TABLEAU")
	s.want(`ECRIS (HASARD 1 2 3)`, "ERR:TROP DE DONNEES POUR HASARD")
	s.want(`ECRIS (TRANCHE [A])`, "ERR:PAS ASSEZ DE DONNEES POUR TRANCHE")
	s.want(`(STOPANIME) (ECRIS)`, "\n")
	// gabarits : une expression, une seule, qui rend une valeur
	s.want(`MONTRE APPLIQUE [1 2] [:? 999]`, "ERR:QUE FAIRE DE 999")
	s.want(`MONTRE APPLIQUE [1 2] [:? * 2]`, "[2 4]\n")
	s.want(`POUR F :A :A FIN`, "ERR:PARAMETRE EN DOUBLE : A")
	s.want(`REPETEPOUR [I 9007199254740992 9007199254740992 1] [ECRIS :I]`, "9007199254740992\n")
	s.want(`REPETEPOUR [I 1 3 1 9] []`, "ERR:REPETEPOUR N'AIME PAS [I 1 3 1 9]")
}

func TestValeurs(t *testing.T) {
	s := newSession(t)
	s.want(`SI PREMIER [TRUE] [ECRIS "OUI] [ECRIS "NON]`, "OUI\n")
	s.want(`MONTRE TRIE [10 1A 2 B 3.5 -1]`, "[-1 2 3.5 10 1A B]\n")
	s.want(`MONTRE TRIE [1A 2 10] MONTRE TRIE [2 10 1A]`, "[2 10 1A]\n[2 10 1A]\n")
	s.want(`ECRIS (HASARD 9223372036854775807 -9223372036854775808)`, "ERR:HASARD N'AIME PAS -9223372036854775808")
	s.want(`ECRIS (HASARD 5 5)`, "5\n")
	s.want(`DONNE "T (TABLEAU 3 5) ECRIS ITEM -9223372036854775806 :T`, "ERR:PAS ASSEZ D'ELEMENTS POUR ITEM")
	s.want(`MONTRE TRANCHE {1 2 3} 2 9223372036854775807`, "{2 3}\n")
	s.want(`FXY (1e308 * 10) - (1e308 * 10) 0`, "ERR:FXY N'AIME PAS NaN")
	s.want(`OCTAVE 4.7`, "ERR:OCTAVE N'AIME PAS 4.7")
	s.want(`ECRIS NOM? [A B]`, "ERR:NOM? N'AIME PAS A B")
	s.want(`FEN LC FXY 1000 0 ENR ECRIS POS`, "-600 0\n")
	s.want(`VE DEMANDE 3 [VE] ECRIS NBTORTUES`, "1\n")
	// profondeurs : erreur Logo, jamais de debordement de pile
	s.want(`DONNE "A TABLEAU 1 REPETE 20000 [DONNE "A LISTEVERSTABLEAU (LISTE :A)] ECRIS COMPTE COPIETABLEAU :A`,
		"ERR:IMBRICATION TROP PROFONDE")
	s.want(`ECRIS TABLEAUMD [`+strings.Repeat("1 ", 70)+`]`, "ERR:TABLEAU TROP GRAND")
	// un booleen range dans une liste ou un tableau reste un booleen
	s.in.SetLang("EN")
	s.want(`MAKE "A ARRAY 1 SETITEM 1 :A TRUE PRINT ITEM 1 :A PRINT FIRST (LIST TRUE)`, "TRUE\nTRUE\n")
	s.want(`PRINT (TABLEAU)`, "ERR:PAS ASSEZ DE DONNEES POUR TABLEAU")
	if got := s.in.ErrorText(errors.New("TROP DE DONNEES POUR HASARD")); got != "TOO MANY INPUTS TO RANDOM" {
		t.Error(got)
	}
}

func TestCalcul(t *testing.T) {
	s := newSession(t)
	s.want(`ECRIS DEVELOPPE [(3X+5Y)(3X-5Y)]`, "9X^2 - 25Y^2\n")
	s.want(`ECRIS FACTORISE [X^3 - 6X^2 + 11X - 6]`, "(X - 1)(X - 2)(X - 3)\n")
	s.want(`ECRIS DEVELOPPE [-X^2 + - - 3]`, "-X^2 + 3\n")
	s.want(`ECRIS DEVELOPPE [(A+B+C+D+E+F+G+H)^100]`, "ERR:CALCUL TROP GROS")
	s.want(`ECRIS EVALUE [((X^1000)^1000)] [X 99999999999999999999]`, "ERR:CALCUL TROP GROS")
	// valeurs interdites : un denominateur simplifie ne doit pas s'oublier
	s.want(`ECRIS EVALUE [(X-1)/(X-1)] [X 2]`, "1\n")
	s.want(`ECRIS EVALUE [(X-1)/(X-1)] [X 1]`, "ERR:DIVISION PAR ZERO")
	s.want(`ECRIS RESOUS [X/X = 1]`, "toujours vrai, sauf pour X = 0\n")
	s.want(`ECRIS RESOUS [(X^2-1)/(X-1) = 2]`, "aucune solution\n")
	s.want(`ECRIS RESOUS [(X^2-3X+2)/(X-1) = X^2 - 2]`, "X = 0\n")
	s.want(`ECRIS RESOUS [(X^2+1)/(X^2+1) = 1]`, "toujours vrai : n'importe quel nombre convient\n")
	s.want(`ECRIS RESOUS [(Y/Y) = X]`, "ERR:JE NE RESOUS QU'UNE EQUATION A UNE SEULE INCONNUE")
}

// --- sauvegarde : ce qui est ecrit doit redonner les memes objets ---

func TestAllerRetour(t *testing.T) {
	s := newSession(t)
	s.ok(`DONNE "W (MOT "A CAR 32 "B CAR 10 "C)
DONNE "L (LISTE "007 "X\]Y "A\;B :W)
DONNE "T {[A B] 2 "07 {1 -2}@-3}
DONNE "N -5.5
DONNE "G 123456789012345678901234567890
DONNE "B FAUX
DONNE "INF 1e308 * 10
DONNE "X (MOT "Y CAR 33)
DONNE "Z (MOT "Y CAR 126)
DONNE (MOT "N CAR 32 "M) 4
DONNE "C [SI :A > -1 [ECRIS "OUI (ECRIS 1 2)] -:X]
POUR P :A ECRIS "BONJOUR ECRIS (SOMME :A -1 2) ECRIS [A "B :C] RENDS -:A FIN`)
	avant := s.run(`IMTOUT`)
	s.ok(`SAUVE "RT CONTENU .EFT RAMENE "RT`)
	if apres := s.run(`IMTOUT`); apres != avant {
		t.Errorf("aller-retour infidele\n--- avant\n%s--- apres\n%s", avant, apres)
	}
	s.want(`MONTRE COMPTE :W MONTRE ITEM 1 :L MONTRE NOMBRE? ITEM 1 :L MONTRE ITEM 3 :T MONTRE COMPTE :X MONTRE COMPTE :Z MONTRE ITEM -3 ITEM 4 :T MONTRE :G + 1 MONTRE :B MONTRE :INF MONTRE CHOSE (MOT "N CAR 32 "M) MONTRE P 3`,
		"5\n007\nVRAI\n07\n2\n2\n1\n123456789012345678901234567891\nFAUX\n+Inf\n4\nBONJOUR\n4\nA B :C\n-3\n")
}

// une procedure dont le corps n'a pas d'ecriture litterale (mot a blanc interne)
func TestSauveCorpsDynamique(t *testing.T) {
	s := newSession(t)
	s.ok(`DONNE "L (PHRASE [ECRIS] (LISTE (MOT "A (MOT CAR 32 "B)))) DEFINIS "P [] :L`)
	s.want(`P`, "A B\n")
	s.ok(`SAUVE "ROUND [P] EFP "P RAMENE "ROUND`)
	s.want(`P`, "A B\n")
	s.want(`MONTRE TEXTE "P`, "[[] [ECRIS A B]]\n")
	// un FIN dans le corps ne tronque pas la definition relue
	s.ok(`DEFINIS "Q [] [ECRIS 1 FIN ECRIS 2] SAUVE "Q [Q] EFP "Q RAMENE "Q`)
	s.want(`MONTRE TEXTE "Q`, "[[] [ECRIS 1 FIN ECRIS 2]]\n")
}

// une liste executable garde la nature de ses elements, meme a cote d'un mot
// qui n'a pas de litteral
func TestSauveListeExecutable(t *testing.T) {
	s := newSession(t)
	s.ok(`DONNE "L (PHRASE [ECRIS] (LISTE (MOT "A (MOT CAR 32 "B))))`)
	s.want(`EXEC :L`, "A B\n")
	s.ok(`SAUVE "ROUND ["L] EFN "L RAMENE "ROUND`)
	s.want(`EXEC :L`, "A B\n")
	s.ok(`DONNE "M (PHRASE [SI :K > 0 [ECRIS "PLUS] ECRIS (SOMME :K 1)] (LISTE VRAI (MOT "U CAR 32)))
SAUVE "M ["M] EFN "M RAMENE "M`)
	s.want(`DONNE "K 3 EXEC SD SD :M`, "PLUS\n4\n")
}

func TestSauveNomsEchappes(t *testing.T) {
	s := newSession(t)
	s.ok(`POUR A\+B :X\-Y ECRIS :X\-Y FIN`)
	s.want(`A\+B 7`, "7\n")
	s.ok(`SAUVE "NAME [A\+B] EFP "A\+B RAMENE "NAME`)
	s.want(`A\+B 8 ECRIS PROC? "A`, "8\nFAUX\n")
	s.want(`IMTS`, "POUR A\\+B :X\\-Y\n")
}

func TestSauveBooleens(t *testing.T) {
	s := newSession(t)
	s.ok(`DONNE "A TABLEAU 2 FIXEITEM 1 :A VRAI FIXEITEM 2 :A (LISTE FAUX "VRAI)
SAUVE "BOOL ["A] EFN "A RAMENE "BOOL`)
	s.in.SetLang("EN")
	// un vrai booleen s'affiche TRUE en anglais, le mot VRAI reste VRAI
	s.want(`PRINT ITEM 1 :A PRINT ITEM 1 ITEM 2 :A PRINT ITEM 2 ITEM 2 :A`, "TRUE\nFALSE\nVRAI\n")
}

// SAUVE ne remplace pas un fichier par une sauvegarde que RAMENE refuserait
func TestSauveRelisible(t *testing.T) {
	s := newSession(t)
	s.ok(`DONNE "OK 1 SAUVE "BIG ["OK] SAUVE "DEEP ["OK]`)
	s.want(`DONNE "W "A REPETE 24 [DONNE "W (MOT :W :W)] SAUVE "BIG ["W]`, "ERR:FICHIER TROP GROS")
	s.want(`DONNE "L [] REPETE 4001 [DONNE "L (LISTE :L)] SAUVE "DEEP ["L]`, "ERR:IMBRICATION TROP PROFONDE")
	s.want(`EFN "OK RAMENE "BIG ECRIS :OK EFN "OK RAMENE "DEEP ECRIS :OK`, "1\n1\n")
	// selection cyclique ou en longue chaine : ni boucle ni debordement
	s.ok(`DONNE "C [:C] SAUVE "CYC [:C]`)
	var chaine strings.Builder
	chaine.WriteString("DONNE \"X 1 DONNE \"V0 [\"X]\n")
	for k := 1; k <= 20000; k++ {
		fmt.Fprintf(&chaine, "DONNE \"V%d [:V%d]\n", k, k-1)
	}
	s.ok(chaine.String())
	s.ok(`SAUVE "CHAINE [:V20000]`)
	s.want(`EFN "X RAMENE "CHAINE ECRIS :X`, "1\n")
}

func TestSauveSelection(t *testing.T) {
	s := newSession(t)
	s.ok(`POUR X ECRIS 1 FIN DONNE "X 5 SAUVE "S1 ["X] SAUVE "S2 [X] SAUVE "S3 CONTENU`)
	lit := func(nom string) string {
		data, err := os.ReadFile(filepath.Join(s.in.workDirPath(), nom+".GLG"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	proc, vari := "POUR X\nECRIS 1\nFIN\n", "DONNE \"X 5\n"
	if lit("S1") != vari || lit("S2") != proc || lit("S3") != proc+vari {
		t.Errorf("S1=%q S2=%q S3=%q", lit("S1"), lit("S2"), lit("S3"))
	}
}

// --- editeur ---

func TestRollbackEditeur(t *testing.T) {
	s := newSession(t)
	ecrit := func(nom, texte string) {
		if err := os.WriteFile(filepath.Join(s.in.workDirPath(), nom), []byte(texte), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s.ok(`POUR P ECRIS "OLD FIN POUR Q ECRIS "OLDQ FIN`)
	// continuation, FIN suivi d'un commentaire, deux definitions sur une ligne,
	// mot-cle echappe : tout doit etre annule quand le texte echoue
	ecrit("BROKEN.GLG", "POUR~\n P\n ECRIS \"NEW\nFIN ; fin de P\nPOUR Q ECRIS \"NEWQ F\\IN POUR N FIN\nUNKNOWN\n")
	s.want(`PIEGE "ERREUR [RAMENE "BROKEN] P Q ECRIS PROC? "N`, "OLD\nOLDQ\nFAUX\n")
	ecrit("GOOD.GLG", "POUR~\n P\n ECRIS \"NEW\nFIN ; fin de P\n")
	s.want(`RAMENE "GOOD P IM "P`, "NEW\nPOUR~\n P\n ECRIS \"NEW\nFIN ; fin de P\n")
}

func TestTraduction(t *testing.T) {
	s := newSession(t)
	s.ok(`POUR PAIR? :N RENDS 0 = RESTE :N 2 FIN`)
	cas := [][2]string{
		{`ECRIS [AVANCE]`, `PRINT [AVANCE]`},
		{`REPETE 4 [AV 50 TD 90]`, `REPEAT 4 [FORWARD 50 RIGHT 90]`},
		{`SI :X > 3 [AV 10] [ECRIS [RE]]`, `IF :X > 3 [FORWARD 10] [PRINT [RE]]`},
		{`SI MEMBRE? "AV [AV TD] [ECRIS "OUI]`, `IF MEMBER? "AV [AV TD] [PRINT "OUI]`},
		{`SI PAIR? :N [AV 10] [RE 10]`, `IF PAIR? :N [FORWARD 10] [BACK 10]`},
		{`SI INCONNU? :N [AV 10] [RE 10]`, `IF INCONNU? :N [AV 10] [RE 10]`},
		{`REPETE 4 [AV 10] CARRE [AV TD]`, `REPEAT 4 [FORWARD 10] CARRE [AV TD]`},
		{"DONNE \"L [AV RE] ; AV\nAV 10;c", "MAKE \"L [AV RE] ; AV\nFORWARD 10;c"},
		{`ECRIS "AV ECRIS :AV ECRIS 2*COMPTE :L`, `PRINT "AV PRINT :AV PRINT 2*COUNT :L`},
		{`TANTQUE [:I < 3] [DONNE "I :I + 1]`, `WHILE [:I < 3] [MAKE "I :I + 1]`},
		{`POURCHAQUE [AV RE] [ECRIS :?]`, `FOREACH [AV RE] [PRINT :?]`},
		{`ECRIS APPLIQUE [1 2] [SOMME :? 1]`, `PRINT MAP [1 2] [SUM :? 1]`},
		{"REPETE 2 ~\n[AV 10]", "REPEAT 2 ~\n[FORWARD 10]"},
		{`DONNE "T {AV RE}`, `MAKE "T {AV RE}`},
	}
	for _, c := range cas {
		if got := s.in.TranslateProgram(c[0], true); got != c[1] {
			t.Errorf("%q\n  obtenu  %q\n  attendu %q", c[0], got, c[1])
		}
	}
}

// les exemples livres : lus, decoupes et traduits sans surprise
func TestExemples(t *testing.T) {
	files, _ := filepath.Glob("../examples/*.GLG")
	if len(files) == 0 {
		t.Fatal("pas d'exemples")
	}
	s := newSession(t)
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src := string(raw)
		data, err := Read(src)
		if err != nil {
			t.Errorf("%s : %v", f, err)
			continue
		}
		blocks := scanProcBlocks(src)
		for k := 0; k+1 < len(data); k++ {
			if data[k].Kind == DSymbol && isProcStart(data[k].Text) {
				if _, ok := blocks[strings.ToUpper(data[k+1].Text)]; !ok {
					t.Errorf("%s : bloc %s non repere", f, data[k+1].Text)
				}
			}
		}
		en := s.in.TranslateProgram(src, true)
		if _, err := Read(en); err != nil {
			t.Errorf("%s : traduction illisible : %v", f, err)
		}
		if again := s.in.TranslateProgram(s.in.TranslateProgram(en, false), true); again != en {
			t.Errorf("%s : traduction instable", f)
		}
	}
}

// --- fichiers ---

func TestFichiers(t *testing.T) {
	s := newSession(t)
	dir := s.in.workDirPath()
	for _, n := range []string{"NUL", "con.txt", "a:b", "x.", "../x", "COM1", "a*b"} {
		s.want(`OUVREECRITURE "`+n, "ERR:NOM DE FICHIER INVALIDE")
	}
	s.want(`SAUVE "NUL []`, "ERR:NOM DE FICHIER INVALIDE")
	s.ok(`OUVREECRITURE "d.txt FIXEECRITURE "d.txt ECRIS "héllo ECRIS "b FIXEECRITURE [] FERME "d.txt`)
	s.want(`OUVRELECTURE "d.txt FIXELECTURE "d.txt ECRIS LISLIGNE ECRIS POSLECTURE ECRIS LISCARS 1 ECRIS FINFICHIER? ECRIS LISLIGNE ECRIS FINFICHIER?`,
		"héllo\n7\nb\nFAUX\n\nVRAI\n")
	// position au milieu d'un caractere : refusee, et le flux ne bouge pas
	s.want(`FIXEPOSLECTURE 2`, "ERR:POSITION INVALIDE")
	s.want(`ECRIS POSLECTURE FIXEPOSLECTURE 1 ECRIS LISCARS 3 ECRIS POSLECTURE`, "9\néll\n5\n")
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		// le meme fichier sous une autre casse
		s.want(`OUVRELECTURE "D.TXT`, "ERR:FICHIER DEJA OUVERT")
		s.want(`FERME "D.TXT MONTRE TOUSOUVERTS`, "[]\n")
	} else {
		s.ok(`FERME "d.txt`)
	}
	// octets invalides : E2 41 42 se lit U+FFFD, A, B
	if err := os.WriteFile(filepath.Join(dir, "bad.txt"), []byte{0xE2, 0x41, 0x42}, 0o644); err != nil {
		t.Fatal(err)
	}
	s.want(`OUVRELECTURE "bad.txt FIXELECTURE "bad.txt ECRIS COMPTE LISCARS 10 FERME "bad.txt`, "3\n")
	s.want(`OUVREAJOUT "d.txt FIXEECRITURE "d.txt FIXEPOSECRITURE 2`, "ERR:POSITION INVALIDE")
	s.want(`FIXEECRITURE [] ECRIS POSECRITURE FERMETOUT`, "-1\n")
	// une ligne plus longue que le plafond, avec ou sans saut final
	long := bytes.Repeat([]byte{'a'}, maxLineBytes+10)
	os.WriteFile(filepath.Join(dir, "long1.txt"), append(long, '\n'), 0o644)
	os.WriteFile(filepath.Join(dir, "long2.txt"), long, 0o644)
	for _, n := range []string{"long1.txt", "long2.txt"} {
		s.want(`OUVRELECTURE "`+n+` FIXELECTURE "`+n+` ECRIS COMPTE LISLIGNE`, "ERR:LIGNE TROP LONGUE")
		s.ok(`FERMETOUT`)
	}
	// sauvegardes : pas sur un flux ouvert, et sans fichier temporaire oublie
	s.want(`DONNE "V 1 SAUVE "S [V] OUVRELECTURE "S.GLG SAUVE "S [V]`, "ERR:FICHIER OUVERT")
	s.want(`DETRUIS "S`, "ERR:FICHIER OUVERT")
	s.ok(`FERMETOUT DONNE "V 2 SAUVE "S [V] DETRUIS "S`)
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".gologo-") {
			t.Errorf("fichier temporaire oublie : %s", e.Name())
		}
	}
	s.want(`ECRIS FICHIERVERSTABLEAU "absent.txt`, "ERR:FICHIER INTROUVABLE")
	s.want(`RAMENE "ABSENT`, "ERR:LECTURE IMPOSSIBLE")
}

// une sauvegarde reprend les droits du fichier qu'elle remplace
func TestSauveDroits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("droits Unix")
	}
	s := newSession(t)
	s.ok(`DONNE "V 1 SAUVE "PRIVE [V]`)
	path := filepath.Join(s.in.workDirPath(), "PRIVE.GLG")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	s.ok(`DONNE "V 2 SAUVE "PRIVE [V]`)
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("droits apres sauvegarde : %v %v", info.Mode().Perm(), err)
	}
}

// un lien (ou une jonction) vers l'exterieur ne fait pas sortir du dossier de travail
func TestConfinement(t *testing.T) {
	s := newSession(t)
	dehors := t.TempDir()
	os.WriteFile(filepath.Join(dehors, "secret.txt"), []byte("secret\n"), 0o644)
	lien := filepath.Join(s.in.workDirPath(), "lien")
	var err error
	if runtime.GOOS == "windows" {
		err = exec.Command("cmd", "/c", "mklink", "/J", lien, dehors).Run()
	} else {
		err = os.Symlink(dehors, lien)
	}
	if err != nil {
		t.Skip("lien impossible :", err)
	}
	if _, err := os.Stat(filepath.Join(lien, "secret.txt")); err != nil {
		t.Skip("le lien ne marche pas :", err)
	}
	s.want(`OUVRELECTURE "lien/secret.txt`, "ERR:LECTURE IMPOSSIBLE")
	s.want(`OUVREECRITURE "lien/pirate.txt`, "ERR:ECRITURE IMPOSSIBLE")
	s.want(`ECRIS FICHIERVERSTABLEAU "lien/secret.txt`, "ERR:LECTURE IMPOSSIBLE")
	if _, err := os.Stat(filepath.Join(dehors, "pirate.txt")); err == nil {
		t.Error("ecriture hors du dossier de travail")
	}
}

// une sortie qui refuse l'ecriture donne une erreur, pas un succes silencieux
type sortieCassee struct{}

func (sortieCassee) Write([]byte) (int, error) { return 0, errors.New("sortie cassee") }

func TestSortieCassee(t *testing.T) {
	in := New(turtle.New(turtle.NewRecorder()), sortieCassee{})
	in.SetWorkDir(t.TempDir())
	for _, src := range []string{`ECRIS 1`, `POUR P FIN`, `IMTS`} {
		if err := in.RunString(src); !errors.Is(err, errEcritureImpossible) {
			t.Errorf("%s : %v", src, err)
		}
	}
}

// --- tortue ---

func TestAnimationEnroulee(t *testing.T) {
	rec := turtle.NewRecorder()
	tt := turtle.New(rec)
	tt.SetField(turtle.Enroule)
	tt.PenUp()
	if err := tt.SetMotion(0, 2000, 0, 10, turtle.Once); err != nil {
		t.Fatal(err)
	}
	steps := 0
	for tt.AnimStep() {
		if steps++; steps > 1000 {
			t.Fatal("l'animation ne s'arrete pas")
		}
	}
	if st := tt.State(); st.X != 400 || st.Y != 0 {
		t.Errorf("x=%v y=%v", st.X, st.Y)
	}
	// BOUCLE crayon baisse : le retour au depart ne trace rien
	tt.SetField(turtle.Clos)
	tt.SetPos(0, 0)
	tt.PenDownState()
	rec.Segments = nil
	tt.SetMotion(0, 30, 0, 10, turtle.Loop)
	for k := 0; k < 3; k++ {
		tt.AnimStep()
	}
	if st := tt.State(); st.X != 0 || len(rec.Segments) != 3 {
		t.Errorf("boucle : x=%v segments=%d", st.X, len(rec.Segments))
	}
}
