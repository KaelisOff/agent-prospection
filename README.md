# Agent de prospection

Un agent pour la box Kaelis qui **trouve des clients par email**, seul, chaque jour de semaine, et qui
s'améliore avec le temps. Il est écrit pour un cabinet, un indépendant ou une petite entreprise qui n'a
pas le temps de prospecter.

> **État : première version (0.1.0), en essai.** Les remarques sont bienvenues dans les *issues*.

## Ce qu'il fait

- **Il trouve les entreprises** de vos cibles (métiers et lieux, dans ses réglages) dans l'**annuaire
  officiel** (SIRENE), puis leur **propre site** par une recherche web, et y relève une **adresse
  générique** (contact@, accueil@, devis@…). Jamais l'adresse d'une personne, jamais une adresse
  trouvée sur un annuaire ou un réseau social.

- **Chaque matin de semaine**, un point en notification (et par email si vous le voulez) : emails
  envoyés, réponses, désinscriptions, erreurs, test en cours, ce qu'il prévoit ensuite. Une réponse
  intéressée vous est signalée tout de suite.
- **Il choisit sa stratégie** : quels métiers et quelles villes viser, quels messages envoyer, à quelle
  heure.
- **Il écrit et améliore ses messages** avec le modèle de la box, et les teste deux par deux (test A/B)
  jusqu'à avoir des chiffres fiables.
- **Il apprend** : au plus une fois par semaine, il fait évoluer sa stratégie, chiffres à l'appui, et
  explique pourquoi.
- **Deux modes, un bouton** (dans ses réglages) : chaque nouveau message attend votre accord, ou
  l'agent écrit et teste seul et vous montre ses messages.

## Ce qu'il ne peut pas faire

Les garde-fous ne sont **pas dans le code de l'agent** : ils sont appliqués par la box, dans la capacité
email. Même un agent modifié ou trompé par un email piégé ne peut pas les franchir.

- **Un seul email par contact**, jamais de relance automatique en boucle.
- **Mentions légales et lien de désinscription** ajoutés par la box à chaque email ; une personne
  désinscrite n'est plus jamais contactée, par aucun agent.
- **Plafond d'envois par jour** fixé par le propriétaire, montée en charge progressive, et **frein
  automatique** si trop d'emails reviennent en erreur ou si les désinscriptions grimpent.
- **Ni la nuit, ni le dimanche.**
- **Budget** fixé par le propriétaire, jamais dépassé.
- **Aucun pistage** : pas de pixel espion, pas de lien traqué. L'agent mesure les réponses, les
  erreurs et les désinscriptions, pas les ouvertures.
- Prospection **entre professionnels** uniquement, sur des adresses professionnelles, avec un message
  en rapport avec leur métier.

Et comme tout agent Kaelis : pas de réseau, pas d'accès aux fichiers ni aux conversations de la box.
Les contacts et les réponses restent dans la box.

## Installer et régler

1. Téléchargez le paquet `.kagent` de la dernière [version](https://github.com/KaelisOff/agent-prospection/releases).
2. Dans le Centre d'applications de votre box, onglet *Agents* : *Installer un agent*, lisez et
   acceptez les permissions.
3. *Configurer* : votre offre ; votre nom, votre société et son adresse postale (la box les écrit en
   bas de chaque email, c'est obligatoire en prospection) ; les métiers visés (codes NAF comme
   `43.22A`, ou mots du nom), les lieux (codes postaux ou départements), le mode, le nombre d'emails
   par jour, et si vous voulez recevoir le point du jour par email.
4. Connectez la messagerie de la box (*Centre d'applications → Plugins → Messagerie*) : votre adresse
   et un mot de passe d'application, de préférence une adresse dédiée.
5. Activez l'agent. Le lendemain matin, il vous propose ses premiers messages.

## Permissions

| Capacité | Pour quoi |
|---|---|
| modèle | écrire les messages, classer les réponses (intéressé, question, pas intéressé) |
| mémoire | entreprises déjà vues, file d'envoi, résultats des tests (4 Mo) |
| horaires | 8 h 30, 10 h, 14 h et 16 h 30, en semaine |
| annuaire des entreprises | trouver les entreprises de vos cibles |
| web (100 recherches par jour) | trouver le site de chaque entreprise et y lire sa page contact |
| notifications | le point du matin, les alertes |
| demandes d'accord | dépasser une limite, changer de cible |
| envoi d'emails (`mail_send`) | envoyer, dans les limites de la box |
| lecture des réponses (`mail_read`) | seulement les réponses à ses propres emails |

La conception détaillée (stratégie, tests A/B, chiffres, questions ouvertes) est dans
[docs/conception.md](docs/conception.md).

## Développer

```sh
make test      # tests, avec une box simulée (annuaire, web, modèle, emails)
make package   # dist/io.github.kaelisoff.prospection-<version>.kagent
make verify    # paquet reproductible
```

## Licence

[Apache 2.0](LICENSE).
