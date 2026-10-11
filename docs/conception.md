# Conception

Document de travail : les chiffres sont des **valeurs proposées**, à confirmer pendant le
développement.

## Qui fait quoi

La règle des agents Kaelis : l'agent **décide**, la box **exécute et protège**.

| | L'agent | La box (capacité email) | Le propriétaire |
|---|---|---|---|
| Cibles (métiers, villes) | propose et ajuste | | valide les nouvelles cibles |
| Messages | écrit, teste, améliore | ajoute identité, mentions légales, désinscription | peut relire et bloquer un message |
| Volume | choisit le rythme sous le plafond | applique le plafond, la montée en charge, le frein | fixe le plafond |
| Horaires | choisit l'heure d'envoi | refuse la nuit et le dimanche | |
| Contacts | | un seul email par contact, liste d'opposition commune à tous les agents | |
| Budget | demande plus si utile | ne le dépasse jamais | fixe le budget |

## Le cycle

**Chaque matin de semaine (8 h 30)**

1. Lire les réponses arrivées depuis la veille et les classer avec le modèle : *intéressé*, *pas
   intéressé*, *désinscription*, *absent / réponse automatique*, *autre*. Les réponses « intéressé »
   sont signalées tout de suite au propriétaire, qui répond lui-même.
2. Mettre à jour les chiffres de chaque cible et de chaque message.
3. Envoyer le **point du jour** (notification, et email au propriétaire).
4. Préparer les envois du jour : choisir les contacts, le message (selon le test en cours), l'heure.

**En journée** : quelques passages d'envoi, étalés, sous le plafond du jour.

**Une fois par semaine au plus** : revoir la stratégie (voir plus bas).

## Le point du jour

Court, lisible sur un téléphone :

- envoyés hier, réponses (dont intéressées), désinscriptions, erreurs ;
- le test en cours : les deux messages, leurs chiffres, « encore N envois avant de conclure » ;
- ce qui est prévu aujourd'hui ;
- une alerte si un frein s'est déclenché.

## Tests A/B

- Deux messages à la fois sur une même cible, contacts répartis au hasard.
- Ce qu'on compare : le **taux de réponses intéressées**, en pénalisant les désinscriptions. On ne
  mesure pas les ouvertures : cela demanderait un pixel espion, que la CNIL soumet à l'accord préalable
  du destinataire, et qui est peu fiable de toute façon.
- On conclut quand chaque message a été envoyé au moins **100 fois** et qu'un des deux a **plus de
  95 % de chances** d'être meilleur (calcul bayésien simple : loi bêta sur le taux de réponse de
  chaque message). Sinon on continue, au plus **4 semaines**, puis on garde le plus court des deux.
- Le gagnant devient la référence ; le modèle écrit un nouveau challenger, en partant de ce qui a
  marché (objet, longueur, accroche, appel à l'action).

## Évolution de la stratégie

Au plus une fois par semaine, l'agent fait le bilan par cible (métier × ville) :

- taux de réponses intéressées et de désinscriptions, sur assez d'envois pour être fiable ;
- il réduit les cibles qui ne répondent pas, renforce celles qui marchent, et peut proposer de
  nouvelles cibles proches ;
- il explique ses choix au propriétaire, chiffres à l'appui.

Ce qui reste dans les limites est appliqué directement ; le reste devient une **demande d'accord**.

## Demandes d'accord

- augmenter le plafond d'envois ou le budget ;
- ajouter une cible nouvelle (métier ou zone) ;
- reprendre après un frein automatique.

Sans réponse avant l'échéance, c'est non. L'agent ne redemande pas la même chose avant une semaine.

## Les freins (dans la box)

Valeurs proposées, réglables par le propriétaire vers le bas, jamais au-delà des maximums de la box :

| Garde-fou | Proposition |
|---|---|
| Plafond quotidien | 20 emails/jour la première semaine, puis le double chaque semaine, jusqu'au plafond fixé par le propriétaire (200 max) |
| Fenêtre d'envoi | du lundi au samedi, 8 h – 19 h |
| Frein « erreurs » | plus de 5 % d'adresses en erreur sur les 100 derniers envois (dès 50 envois) → pause |
| Frein « désinscriptions » | plus de 2 % sur les 100 derniers envois (dès 50 envois) → pause |
| Contact | un seul email de prospection par adresse, pour toujours |

Ces garde-fous sont en place dans la box (service mail) ; ils s'appliquent à tout agent qui envoie des
emails, pas seulement à celui-ci.

## Questions ouvertes

- **D'où viennent les contacts ?** De l'annuaire officiel (SIRENE, outil `companies` de la box) pour
  les entreprises, puis de leur propre site (outil `web`) pour l'adresse : uniquement une adresse
  générique publiée par l'entreprise elle-même. Les entreprises non diffusibles sont écartées par la
  box, les entrepreneurs individuels par défaut.
- **Quelle boîte d'envoi ?** Pour commencer, la messagerie du propriétaire (SMTP/IMAP), de préférence
  une adresse dédiée ; un service d'envoi dédié, meilleur pour la délivrabilité, pourra s'y ajouter.
- **Répondre depuis l'email** : valider une demande d'accord en répondant au point du jour demande
  d'authentifier la réponse (pas seulement l'adresse d'expéditeur). À concevoir ; en attendant, la
  validation passe par l'application.
