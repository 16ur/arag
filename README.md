# arag

arag est un navigateur de fichiers WebDAV en terminal, écrit en Go avec
Bubble Tea V2. Il permet de parcourir un serveur distant et d'ouvrir un média
dans un lecteur externe.

Le projet vise à fonctionner avec tout serveur WebDAV standard. Seedhost est
le premier preset pris en charge, car il constitue le cas d'usage initial du
projet.

## État du projet

arag est en cours de construction. Le MVP doit permettre de :

- se connecter à un serveur WebDAV ;
- parcourir les dossiers sans charger toute l'arborescence ;
- naviguer avec les flèches ou `hjkl` ;
- ouvrir une vidéo dans IINA après confirmation.

IINA est le lecteur ciblé par le MVP, mais l'architecture permettra d'ajouter
d'autres lecteurs, notamment VLC.

Consulter la [roadmap](docs/ROADMAP.md) pour le périmètre prévu.
