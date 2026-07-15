# WebDAV

## URL

L'URL est configurable afin de prendre en charge tout serveur WebDAV standard.

Preset Seedhost :

`https://mud.seedhost.eu/<user>/webdav`

Le preset fournit un modèle d'URL mais ne modifie pas le protocole ou le
parsing.

## Authentification

HTTP Basic.

Pour le MVP, le mot de passe est demandé sans affichage au démarrage et gardé
uniquement en mémoire pendant la session. Il n'est donc pas nécessaire de le
ressaisir pour chaque dossier ou chaque vidéo.

Une variable d'environnement peut fournir le secret dans un environnement
automatisé. Elle ne doit pas être affichée ou journalisée. Le nom exact de la
variable sera fixé lors de l'implémentation de la configuration.

Le fichier de configuration ne contient jamais le mot de passe en clair. Le
stockage dans le trousseau système est prévu après le MVP.

## Navigation

Méthode : `PROPFIND`.

Header : `Depth: 1`.

Réponse attendue : `207 Multi-Status`.

Les dossiers sont identifiés par `<D:collection/>`.

Les fichiers possèdent :

- `getcontentlength` ;
- `creationdate` ;
- `getlastmodified` lorsque disponible ;
- `getetag`.

Le parseur utilise les espaces de noms XML et ne dépend pas du préfixe `D`.
Les propriétés peuvent être absentes. Le client ignore l'entrée représentant
le dossier interrogé lui-même et retourne des objets Go à l'UI.

Le client ne parse jamais de HTML et n'utilise pas de page d'index comme
solution de repli.

## Réseau

Valeurs par défaut proposées :

- 10 secondes pour la connexion et les en-têtes ;
- 30 secondes pour terminer une requête `PROPFIND` de navigation.

Une requête est annulée lorsque l'utilisateur quitte ou lorsqu'une nouvelle
navigation rend son résultat obsolète. Une erreur de timeout est présentée
séparément d'un refus d'authentification ou d'une réponse XML invalide.

Ces limites sont configurables. Elles concernent la navigation WebDAV, pas la
durée de lecture d'un média dans le lecteur externe.
