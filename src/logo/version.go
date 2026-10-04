package logo

// Version de GoLogo, partagee par tout le programme (bandeau d'accueil, titre de
// fenetre...). Les scripts de packaging la lisent ici ; elle figure aussi dans
// tools/build/versioninfo.json (ressource Windows) et dist/windows/gologo.iss, que
// le workflow de release compare a celle-ci avant de construire quoi que ce soit.
const Version = "2.2"
