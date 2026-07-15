# ADR 001

Navigation lazy.

Pourquoi ?

Une seedbox peut contenir plusieurs milliers de fichiers.

Conséquence

Le TUI ne charge jamais toute l'arborescence.

---

# ADR 002

WebDAV plutôt que SSH.

Pourquoi ?

Un seul protocole.

Pas de parsing HTML.

---

# ADR 003

Client WebDAV généraliste avec presets.

Pourquoi ?

Le cœur du produit doit fonctionner avec tout serveur WebDAV standard. Un
preset, comme Seedhost, simplifie la configuration d'un fournisseur sans
introduire de comportement spécifique dans le client WebDAV.

Conséquence

L'URL WebDAV reste configurable. Un preset ne fournit que des valeurs par
défaut et peut être remplacé par la configuration utilisateur.

---

# ADR 004

Lecteur externe interchangeable.

Pourquoi ?

IINA répond au besoin initial sur macOS, mais ne doit pas limiter arag à une
seule application ou plateforme.

Conséquence

Le MVP implémente IINA derrière un contrat de player minimal. VLC et d'autres
lecteurs pourront être ajoutés après le MVP.

---

# ADR 005

Les secrets ne sont pas stockés en clair.

Pourquoi ?

Demander le mot de passe avant chaque lecture dégrade fortement l'expérience,
mais l'enregistrer dans le fichier de configuration n'est pas acceptable.

Conséquence

Pour le MVP, le mot de passe est demandé sans affichage une seule fois au
démarrage puis conservé uniquement en mémoire pendant la session. Une variable
d'environnement peut être utilisée pour les environnements automatisés. Une
version ultérieure pourra utiliser le trousseau sécurisé du système pour une
persistance consentie par l'utilisateur.

Le mot de passe ne doit apparaître ni dans les logs, ni dans les erreurs, ni
dans l'historique du shell.

---

# ADR 006

Timeouts réseau courts et configurables.

Pourquoi ?

Un TUI doit signaler rapidement un serveur indisponible sans abandonner trop
tôt une seedbox lente.

Conséquence

Valeurs initiales proposées :

- 10 secondes pour établir la connexion et recevoir les en-têtes ;
- 30 secondes maximum pour une requête de navigation `PROPFIND` ;
- annulation immédiate lorsque l'utilisateur quitte ou remplace la navigation
  en cours.

Ces valeurs sont configurables. La lecture du média n'utilise pas le timeout
de navigation, car sa durée est gérée par le lecteur externe.
