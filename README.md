# Ragnaros

Petit outil CLI Linux qui garde en mémoire les deux dernières minutes d'activité
de la machine. Quand le CPU reste au-dessus de 90 % pendant 5 secondes, il
enregistre cet historique dans un fichier pour comprendre ce qui a précédé le pic.

Go 1.23+, bibliothèque standard uniquement, données lues directement dans `/proc`.

## Fonctionnement

Chaque seconde, `ragnaros watch` :

1. lit `/proc/stat` et calcule l'utilisation CPU globale (0 à 100 %) à partir de
   la différence avec la mesure précédente ;
2. lit `MemAvailable` dans `/proc/meminfo` ;
3. lit `/proc/<pid>/stat` de chaque processus et garde les 5 plus gourmands en CPU
   (100 % = un cœur entier, donc un processus peut dépasser 100 %) ;
4. ajoute la mesure à un historique limité à 120 mesures (2 minutes) ;
5. si le CPU dépasse 90 % sur 5 mesures consécutives, écrit l'historique dans
   `incidents/incident-AAAAMMJJ-HHMMSS.jsonl` (une mesure JSON par ligne).
   Un pic prolongé ne produit qu'un seul fichier. La détection se réarme dès que
   le CPU repasse à 90 % ou moins.

`ragnaros report` relit un fichier d'incident et affiche la période, le pic CPU,
le minimum de mémoire disponible et les processus les plus actifs.

## Commandes

```sh
go build -o ragnaros .
./ragnaros watch                                  # Ctrl+C pour arrêter
./ragnaros report incidents/incident-20240906-172008.jsonl
go test ./...
```

Le dossier `incidents/` est créé dans le répertoire courant.

## Exemple de sortie

```
$ ./ragnaros watch
Ragnaros is watching. Press Ctrl+C to stop.

17:16:21  CPU   6.2 %  available memory 44.0 GiB
     2491  kwin_wayland       23.1 %
     4141  chrome             20.1 %
     4199  chrome             11.1 %
   180255  chrome             11.1 %
   179450  chrome              8.0 %
^C
Stopped.
```

```
$ ./ragnaros report incidents/incident-20240906-172008.jsonl
Incident file: incidents/incident-20240906-172008.jsonl
Period:           2024-09-06 17:20:00 -> 2024-09-06 17:20:11 (11s, 12 samples)
CPU peak:         98.0 % at 17:20:08
Lowest memory:    22.9 GiB available at 17:20:11

Most active processes (ranked by CPU used, SEEN = times in the top 5):
      PID  NAME                  SEEN   MAX CPU
     9001  cc1plus               7/12  1400.0 %
     4141  chrome               12/12    31.0 %

Process CPU: 100 % means one full core.
These are observations of what was running before the spike, not a proof of its cause.
```

## Limites

- Linux uniquement (`/proc`). Pas de température pour l'instant (V2).
- Le rapport montre des observations, pas une cause prouvée.
- Un processus apparu depuis moins d'une seconde n'est pas encore classé : il faut
  deux lectures pour calculer son CPU. Idem pour un PID réutilisé.
- Un processus très court (moins d'une seconde) peut passer inaperçu.
- Seuls les 5 premiers processus de chaque seconde sont enregistrés.
- Le nom vient de `/proc/<pid>/stat` : 15 caractères maximum, sans arguments.
- L'historique n'existe qu'en mémoire : ce qui précède le lancement de `watch`
  n'est pas connu, et un Ctrl+C pendant un pic n'enregistre rien.
- Seuil (90 %), durée (5 s) et taille d'historique (120) sont des constantes dans
  `watch.go`, sans option en ligne de commande.
