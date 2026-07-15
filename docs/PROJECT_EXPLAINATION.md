# arag

arag est un navigateur WebDAV généraliste permettant de parcourir des fichiers
distants depuis un terminal et de lancer rapidement des vidéos dans un lecteur
externe.

Seedhost est le premier preset fourni, car il constitue le cas d'usage initial.
Un preset renseigne des valeurs adaptées à un fournisseur sans enfermer le
client dans ce fournisseur. Une configuration WebDAV personnalisée reste
toujours possible.

## Objectifs

- Naviguer dans les fichiers de tout serveur WebDAV compatible.
- Proposer un preset Seedhost, puis d'autres presets selon les besoins.
- Interface TUI avec Bubble Tea V2.
- Ouverture des vidéos dans un lecteur externe.
- Utiliser IINA pour le MVP, sans le rendre obligatoire dans l'architecture.
- Navigation fluide au clavier avec "hjkl" et les flèches directionnelles.

## Principes produit

- Le produit doit rester utilisable sans connaître Vim : les flèches et une
  aide intégrée sont toujours disponibles.
- Une information importante ne doit jamais être communiquée uniquement par
  une couleur.
- Les erreurs doivent expliquer leur cause et, lorsque possible, comment les
  corriger.
- Le chargement d'un dossier doit rester progressif et annulable.
- Le mot de passe ne doit jamais être enregistré en clair.

## Hors du MVP

- recherche ;
- historique de navigation persistant ;
- prise en charge officielle de plusieurs lecteurs et plateformes ;
- stockage persistant des secrets dans les trousseaux système.
