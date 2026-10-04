//go:build unix

package logo

import "syscall"

// drapeau ajoute a toutes les ouvertures de fichiers de donnees : sans effet sur un
// fichier ordinaire, mais l'ouverture d'un tube nomme (FIFO) rend la main tout de
// suite au lieu d'attendre qu'un autre processus l'ouvre a son tour
const openNonBlock = syscall.O_NONBLOCK
