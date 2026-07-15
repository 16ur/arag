# Projet

arag est un navigateur WebDAV.

Stack

- Go
- Bubble Tea V2

Objectif

Créer un navigateur de fichiers distant.

Portée produit

- Compatible avec les serveurs WebDAV standards.
- Seedhost est le premier preset, pas une dépendance du client.
- IINA est le lecteur du MVP, mais le player doit rester interchangeable.
- La recherche et l'historique persistant sont hors du MVP.

Architecture

UI -> WebDAV

UI -> Player -> IINA pour le MVP

Contraintes

Ne jamais parser du HTML.

Toujours utiliser WebDAV.

Ne jamais lancer une requête réseau dans View().

Ne jamais effectuer de logique métier dans View().

Préférer de petites fonctions.

Toujours documenter les packages publics.

Éviter les dépendances si la bibliothèque standard suffit.

Ne jamais stocker ou journaliser un secret en clair.

Une information importante ne doit jamais dépendre uniquement de la couleur.

Conserver les flèches directionnelles en plus des raccourcis hjkl.
