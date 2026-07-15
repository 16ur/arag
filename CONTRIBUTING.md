# Règles

Toujours préférer la simplicité.

Pas d'abstraction prématurée.

Pas de dépendance inutile.

Chaque package possède une responsabilité claire.

Éviter les fonctions > 100 lignes.

Privilégier des noms explicites.

## Contraintes produit

- conserver la compatibilité avec les serveurs WebDAV génériques ;
- implémenter les particularités des fournisseurs sous forme de presets ;
- ne jamais enregistrer ou journaliser un secret en clair ;
- ne jamais lancer de requête réseau ou de logique métier dans `View()` ;
- accompagner tout nouveau parsing WebDAV de tests XML ;
- conserver les flèches directionnelles lorsque des raccourcis `hjkl` sont
  ajoutés ;
- ne pas communiquer une information uniquement par la couleur.
