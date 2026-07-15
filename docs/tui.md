# Interface TUI

Au démarrage, arag charge et affiche le contenu du serveur WebDAV configuré.
Avec le preset Seedhost, ce contenu correspond à la racine WebDAV de la
seedbox.

## Navigation

- flèches haut/bas ou `j`/`k` : déplacer la sélection ;
- `Entrée` ou `l` : entrer dans un dossier ou sélectionner un fichier ;
- flèche gauche, `h` ou `Retour arrière` : revenir au dossier parent ;
- `?` : afficher ou masquer l'aide ;
- `q` : quitter.

Les raccourcis réellement implémentés doivent rester visibles dans l'aide. Les
flèches sont toujours prises en charge afin que l'interface reste accessible
aux personnes ne connaissant pas Vim.

## Ouverture d'un média

Pour un fichier vidéo pris en charge, `Entrée` affiche une confirmation avant
de l'ouvrir dans IINA. La confirmation indique clairement le nom du fichier et
permet d'annuler sans effet de bord.

## États à afficher

- chargement en cours ;
- dossier vide ;
- contenu disponible ;
- demande de confirmation ;
- erreur récupérable.

Les erreurs distinguent au minimum :

- configuration ou URL invalide ;
- identifiants refusés ;
- serveur indisponible ou timeout ;
- réponse WebDAV invalide ;
- lecteur absent ou impossible à lancer.

Chaque erreur indique une action possible : réessayer, vérifier la
configuration, ressaisir le mot de passe ou revenir au dossier précédent.

## Accessibilité et affichage

- aucune information ne dépend uniquement de la couleur ;
- la sélection possède également un marqueur textuel ;
- les noms longs sont tronqués proprement sans casser la mise en page ;
- les petits terminaux conservent la navigation et les messages essentiels ;
- le redimensionnement du terminal ne provoque ni requête réseau ni perte de
  sélection ;
- les messages restent compréhensibles sans jargon WebDAV inutile.
