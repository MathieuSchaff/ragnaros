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
3. lit la température la plus haute parmi `/sys/class/thermal/thermal_zone*/temp`
   (ignorée si aucun capteur n'est lisible) ;
4. lit `/proc/<pid>/stat` de chaque processus et garde les 5 plus gourmands en CPU
   (100 % = un cœur entier, donc un processus peut dépasser 100 %), avec leur
   ligne de commande (`/proc/<pid>/cmdline`) ;
5. ajoute la mesure à un historique limité à 120 mesures (2 minutes) ;
6. si le CPU dépasse 90 % sur 5 mesures consécutives, écrit l'historique dans
   `incidents/incident-AAAAMMJJ-HHMMSS.jsonl` (une mesure JSON par ligne).
   Un pic prolongé ne produit qu'un seul fichier. La détection se réarme dès que
   le CPU repasse à 90 % ou moins.

Un Ctrl+C pendant un pic pas encore confirmé enregistre quand même l'historique :
on arrête souvent l'outil justement parce que la machine rame.

`ragnaros report` relit un fichier d'incident et affiche la période, le pic CPU,
le minimum de mémoire disponible, la température maximale et les processus les
plus actifs (classés par CPU cumulé sur la période).

## Commandes

```sh
go build -o ragnaros .
./ragnaros watch                                  # Ctrl+C pour arrêter
./ragnaros watch -threshold 70 -samples 10 -dir /tmp/ragnaros
./ragnaros watch -threshold 1 -samples 2          # déclenche un incident sans charger la machine
./ragnaros report incidents/incident-20240906-172008.jsonl
go test ./...
```

Par défaut, le dossier `incidents/` est créé dans le répertoire courant.

## Exemple de sortie

```
$ ./ragnaros watch
Ragnaros is watching. Press Ctrl+C to stop.

17:24:42  CPU   4.4 %  available memory 44.0 GiB  max temperature 55 °C
     4199    16.1 %  /opt/google/chrome/chrome --type=gpu-process --gpu-recent...
     2491    13.1 %  /usr/bin/kwin_wayland --wayland-fd 7 --socket wayland-0 -...
   180255    11.1 %  /opt/google/chrome/chrome --type=renderer --crashpad-hand...
   179450     7.1 %  /opt/google/chrome/chrome --type=renderer --crashpad-hand...
      154     6.0 %  irq/9-acpi
^C
Stopped.
```

```
$ ./ragnaros report incidents/incident-20240906-172008.jsonl
Incident file: incidents/incident-20240906-172008.jsonl
Period:           2024-09-06 17:20:00 -> 2024-09-06 17:20:11 (11s, 12 samples)
CPU peak:         98.0 % at 17:20:08
Lowest memory:    22.9 GiB available at 17:20:11
Max temperature:  92 °C at 17:20:10

Most active processes (ranked by CPU used, SEEN = times in the top 5):
      PID    SEEN   MAX CPU  COMMAND
     9001    7/12  1400.0 %  /usr/libexec/gcc/x86_64-linux-gnu/13/cc1plus -quiet main.cpp
     4141   12/12    31.0 %  /opt/google/chrome/chrome

Process CPU: 100 % means one full core.
These are observations of what was running before the spike, not a proof of its cause.
```

## Limites

- Linux uniquement (`/proc`, `/sys`).
- La température est celle de la zone thermique la plus chaude, quel que soit le
  composant (CPU, Wi-Fi, ...). Les capteurs exposés seulement par `hwmon`
  (fréquent sur AMD avec `k10temp`) ne sont pas lus.
- Le rapport montre des observations, pas une cause prouvée.
- Un processus très court (né et terminé entre deux mesures) passe inaperçu.
- Un processus absent de la mesure précédente est compté depuis zéro. Si sa lecture
  avait échoué alors qu'il existait déjà, tout son temps CPU passé serait attribué
  à la dernière seconde (cas rare).
- Seuls les 5 premiers processus de chaque seconde sont enregistrés.
- La ligne de commande est coupée à 60 caractères.
- L'historique n'existe qu'en mémoire : ce qui précède le lancement de `watch`
  n'est pas connu, et un arrêt brutal (`kill -9`, crash) n'enregistre rien.
- La taille d'historique (120 mesures) et l'intervalle (1 s) sont des constantes
  dans `watch.go`.
