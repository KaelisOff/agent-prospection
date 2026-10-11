# Agent de prospection

Un agent pour la box Kaelis qui **trouve des clients par email**, seul, chaque jour de semaine, et qui
s'améliore avec le temps. Il est écrit pour un cabinet, un indépendant ou une petite entreprise qui n'a
pas le temps de prospecter.

> **État : en conception.** La capacité email de la box, qui porte les garde-fous, existe depuis le
> kit [v0.2.0](https://github.com/KaelisOff/agent-sdk-go) ; le code de l'agent arrive ensuite. Ce dépôt
> décrit ce que fera l'agent : les remarques sont bienvenues dans les *issues*.

## Ce qu'il fait

- **Chaque matin de semaine**, un point en notification (et par email) : emails envoyés, réponses,
  désinscriptions, erreurs, test en cours, ce qu'il prévoit ensuite.
- **Il choisit sa stratégie** : quels métiers et quelles villes viser, quels messages envoyer, à quelle
  heure.
- **Il écrit et améliore ses messages** avec le modèle de la box, et les teste deux par deux (test A/B)
  jusqu'à avoir des chiffres fiables.
- **Il apprend** : au plus une fois par semaine, il fait évoluer sa stratégie, chiffres à l'appui, et
  explique pourquoi.
- **Il demande votre accord** dès qu'il voudrait dépasser une limite (plus d'envois, plus de budget,
  une nouvelle cible) : une réponse en un clic depuis l'application.

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

## Permissions prévues

| Capacité | Pour quoi |
|---|---|
| modèle | écrire les messages, lire les réponses (intéressé, pas intéressé, absent…), préparer les points |
| mémoire | contacts déjà traités, résultats des tests, stratégie en cours |
| horaires | un passage le matin et quelques passages d'envoi en journée, en semaine |
| notifications | le point du matin, les alertes |
| demandes d'accord | dépasser une limite, changer de cible |
| envoi d'emails (`mail_send`) | envoyer, dans les limites de la box |
| lecture des réponses (`mail_read`) | seulement les réponses à ses propres emails |

La conception détaillée (stratégie, tests A/B, chiffres, questions ouvertes) est dans
[docs/conception.md](docs/conception.md).

## Licence

[Apache 2.0](LICENSE).
