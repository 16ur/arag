# Architecture

Le projet est composé de quatre responsabilités principales.

## UI

Bubble Tea V2.

Responsabilités :

- affichage
- navigation
- raccourcis clavier
- états de chargement, de confirmation et d'erreur

La UI ne connaît pas le XML WebDAV.

`View()` produit uniquement une représentation de l'état courant. Elle ne
lance aucune requête réseau et ne contient aucune logique métier. Les effets de
bord sont exécutés par des commandes Bubble Tea.

---

## WebDAV

Responsabilités :

- authentification
- requêtes PROPFIND
- parsing XML
- normalisation des URL et chemins
- application des timeouts

Le client retourne des objets Go.

La navigation utilise `Depth: 1`. Le client ne charge jamais toute
l'arborescence distante.

---

## Player

Responsabilités :

- ouvrir une URL dans un lecteur externe ;
- adapter l'appel au lecteur sélectionné.

Le player ne connaît pas Bubble Tea.

IINA est la première implémentation du MVP. Le contrat du package ne doit pas
dépendre d'IINA afin de permettre l'ajout ultérieur de VLC ou d'un autre
lecteur.

La transmission de l'authentification au lecteur devra éviter d'exposer les
identifiants dans les logs, les messages d'erreur ou un fichier. Ce point doit
faire l'objet d'un test technique avant de finaliser l'intégration IINA.

---

## Configuration

Responsabilités :

- charger l'URL, le preset, l'utilisateur et le lecteur ;
- appliquer les valeurs par défaut ;
- valider la configuration sans effectuer de requête réseau.

La configuration non sensible pourra être conservée dans un fichier. Les
secrets n'y sont jamais écrits en clair.

## Flux principal

1. La configuration prépare un client WebDAV.
2. Une commande Bubble Tea demande le contenu du dossier courant.
3. Le client exécute `PROPFIND` et retourne des objets Go.
4. `Update()` intègre le résultat dans l'état de l'UI.
5. `View()` affiche cet état.
6. Lorsqu'un fichier est confirmé, le player l'ouvre avec le lecteur choisi.
