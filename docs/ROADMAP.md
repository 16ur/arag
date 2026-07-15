# MVP

- [ ] configuration d'une URL WebDAV personnalisée
- [ ] preset Seedhost
- [ ] authentification HTTP Basic avec saisie masquée une fois par session
- [ ] connexion WebDAV avec timeout et annulation
- [ ] parser les réponses XML de PROPFIND
- [ ] normaliser les URL et chemins retournés par le serveur
- [ ] navigation lazy avec `Depth: 1`
- [ ] navigation avec les flèches et `hjkl`
- [ ] retour au dossier parent
- [ ] états de chargement, vide et erreur
- [ ] aide intégrée et raccourcis découvrables
- [ ] affichage adapté aux petits terminaux et aux noms longs
- [ ] confirmation avant l'ouverture d'une vidéo
- [ ] ouverture dans IINA sans exposer les identifiants
- [ ] tests unitaires du parseur XML
- [ ] tests d'intégration avec un faux serveur WebDAV local
- [ ] documentation d'installation et de configuration

---

# V1

- [ ] choix du lecteur externe
- [ ] prise en charge de VLC
- [ ] stockage optionnel dans le trousseau sécurisé du système
- [ ] presets WebDAV supplémentaires
- [ ] binaires précompilés pour les plateformes prises en charge
- [ ] recherche dans le dossier courant
- [ ] historique de navigation persistant

## Critères de qualité

- [ ] aucune requête réseau ou logique métier dans `View()`
- [ ] aucun secret dans les fichiers, logs ou messages d'erreur
- [ ] aucune information transmise uniquement par la couleur
- [ ] erreurs distinctes pour URL invalide, authentification refusée, serveur
      indisponible et réponse WebDAV invalide
- [ ] navigation utilisable sans connaître les raccourcis Vim
