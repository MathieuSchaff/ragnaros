# Ragnaros

Petit outil CLI Linux qui garde en mémoire les deux dernières minutes d'activité
de la machine. Quand la machine reste chargée pendant 5 secondes (CPU global, un
cœur saturé ou température), il enregistre cet historique dans un fichier pour
comprendre ce qui a précédé le pic.

Go 1.23+, bibliothèque standard uniquement. Données lues dans `/proc` et `/sys`,
plus `nvidia-smi` s'il est installé.

## Fonctionnement

Chaque seconde (réglable avec `-interval`), `ragnaros watch` :

1. lit `/proc/stat` et calcule l'utilisation CPU globale (0 à 100 %) à partir de
   la différence avec la mesure précédente ;
2. lit la charge moyenne sur 1 minute dans `/proc/loadavg` : nombre moyen de
   tâches qui tournent ou attendent (CPU ou disque), à comparer au nombre de cœurs ;
3. lit `MemAvailable` et le swap utilisé (`SwapTotal - SwapFree`) dans `/proc/meminfo` ;
4. lit la température du CPU via `hwmon` (`coretemp` pour Intel, `k10temp` ou
   `zenpower` pour AMD), sinon la zone la plus chaude de
   `/sys/class/thermal/thermal_zone*/temp` (ignorée si aucun capteur n'est lisible) ;
5. lit `/proc/<pid>/stat` de chaque processus et garde les 5 plus gourmands en CPU
   (100 % = un cœur entier, donc un processus peut dépasser 100 %), avec leur
   ligne de commande (`/proc/<pid>/cmdline`) et la fréquence du cœur sur lequel
   ils tournent (champ 39 de `stat`) ;
6. lit la fréquence de chaque cœur dans `/sys/devices/system/cpu/cpu*/cpufreq` :
   un cœur chargé loin de sa fréquence maximale signale souvent un throttling
   thermique ;
7. lit le GPU NVIDIA (utilisation, température, consommation) avec `nvidia-smi`,
   sauf si la dGPU dort (`power/runtime_status` = `suspended`) : l'appeler la
   réveillerait, et une dGPU réveillée chauffe le CPU par le heatpipe partagé ;
8. ajoute la mesure à un historique qui couvre 2 minutes (120 mesures à 1 s) ;
9. si 5 mesures consécutives sont « hautes », écrit l'historique dans
   `incidents/incident-AAAAMMJJ-HHMMSS.jsonl` (une mesure JSON par ligne) et
   affiche la raison. Une mesure est haute si au moins un de ces cas est vrai :
   - CPU global au-dessus de 90 % (`-threshold`) ;
   - un processus au-dessus de 95 % d'un cœur (`-core-threshold`) : un programme
     mono-thread, comme beaucoup de jeux, rame alors que le CPU global reste à 5 % ;
   - température au-dessus de 90 °C (`-temp-threshold`).

   Un pic prolongé ne produit qu'un seul fichier. La détection se réarme dès
   qu'une mesure n'est plus haute.

Un Ctrl+C pendant un pic pas encore confirmé enregistre quand même l'historique :
on arrête souvent l'outil justement parce que la machine rame.

`ragnaros report` relit un fichier d'incident et affiche la période, le pic CPU,
le pic de charge, le minimum de mémoire disponible, le maximum de swap utilisé,
la température maximale, le moment où le cœur le plus rapide était le plus lent,
le résumé GPU et les processus les plus actifs (classés par CPU cumulé sur la
période, avec la fréquence moyenne de leur cœur).

## Commandes

```sh
go build -o ragnaros .
./ragnaros watch                                  # Ctrl+C pour arrêter
./ragnaros watch -threshold 70 -samples 10 -dir /tmp/ragnaros
./ragnaros watch -core-threshold 0 -temp-threshold 0   # seulement le CPU global
./ragnaros watch -interval 500ms                  # historique : 240 mesures
./ragnaros watch -h                               # liste des options
./ragnaros watch -threshold 1 -samples 2          # déclenche un incident sans charger la machine
./ragnaros report incidents/incident-20240906-172008.jsonl
go test ./...
```

Par défaut, le dossier `incidents/` est créé dans le répertoire courant.

## Exemple de sortie

```
$ ./ragnaros watch
Ragnaros is watching. Press Ctrl+C to stop.

17:37:38  CPU   6.3 %  load 2.37  available memory 43.4 GiB  swap used 0 MiB  temperature 55 °C  fastest core 2250/4700 MHz  GPU 14 % 53 °C 17 W
   200743    27.2 %    687/4600 MHz  claude
     2491    18.1 %   1817/4600 MHz  /usr/bin/kwin_wayland --wayland-fd 7 --socket wayland-0 -...
     4199    12.1 %    946/4600 MHz  /opt/google/chrome/chrome --type=gpu-process --gpu-recent...
   180255     8.1 %   1256/3500 MHz  /opt/google/chrome/chrome --type=renderer --crashpad-hand...
     3887     8.1 %    664/3500 MHz  /usr/bin/ghostty --gtk-single-instance=true --initial-win...
^C
Stopped.
```

Rapport illustratif, construit d'après les relevés d'un raid WoW
(`dev-setup/cheatsheet/octowow-perf.md`) : CPU global bas, un cœur saturé,
fréquence effondrée, 91 °C.

```
$ ./ragnaros report incidents/incident-20240906-212008.jsonl
Incident file: incidents/incident-20240906-212008.jsonl
Period:           2024-09-06 21:18:09 -> 2024-09-06 21:20:08 (1m59s, 120 samples)
CPU peak:         9.1 % at 21:20:01
Load peak (1 m):  2.80 at 21:20:05
Lowest memory:    41.2 GiB available at 21:19:58
Max swap used:    none
Max temperature:  91 °C at 21:20:04
Slowest moment:   fastest core at 2697 MHz (limit 4700) at 21:20:03
GPU peak:         14 % at 21:19:40, max 83 °C, max 24 W (asleep in 0/120 samples)

Most active processes (ranked by CPU used, SEEN = times in the top 5):
      PID    SEEN   MAX CPU        AVG CORE  COMMAND
   412345 120/120   100.3 %   1650/4700 MHz  Z:\OctoWoW\WoW.exe
     2491 120/120    21.0 %   1800/4600 MHz  /usr/bin/kwin_wayland --wayland-fd 7 --socket wayland-0 -...

Process CPU: 100 % means one full core. AVG CORE: mean speed of the core it ran on.
These are observations of what was running before the spike, not a proof of its cause.
```

## Limites

- Linux uniquement (`/proc`, `/sys`).
- Sans capteur CPU `hwmon` reconnu, la température est celle de la zone thermique
  la plus chaude, qui peut être un autre composant (Wi-Fi, batterie...).
- Le rapport montre des observations, pas une cause prouvée.
- Un processus très court (né et terminé entre deux mesures) passe inaperçu.
- Un processus absent de la mesure précédente est compté depuis zéro. Si sa lecture
  avait échoué alors qu'il existait déjà, tout son temps CPU passé serait attribué
  à la dernière seconde. Une valeur au-delà de tous les cœurs est écartée, une
  valeur plausible mais fausse reste possible (cas rare).
- Seuls les 5 premiers processus de chaque seconde sont enregistrés.
- La ligne de commande est coupée à 60 caractères.
- L'historique n'existe qu'en mémoire : ce qui précède le lancement de `watch`
  n'est pas connu, et un arrêt brutal (`kill -9`, crash) n'enregistre rien.
- La durée d'historique (2 minutes) est une constante dans `watch.go`.
- Le déclencheur « un cœur saturé » se déclenche une fois au début d'une longue
  session de jeu, puis plus jusqu'à ce que le processus redescende sous le seuil.
  Une compilation déclenche aussi.
- `nvidia-smi` est lancé à chaque mesure quand la dGPU est éveillée : environ
  50 ms, et il apparaît parfois dans le top. Seul le premier GPU NVIDIA est lu.
  Les GPU AMD et Intel ne sont pas lus.
- La fréquence d'un processus est celle du cœur où il a tourné en dernier, lue au
  moment de la mesure : une approximation.
- La raison du déclenchement est affichée par `watch` mais pas écrite dans le fichier.
- La charge moyenne sur 1 minute réagit lentement : elle monte encore après le
  début du pic.
