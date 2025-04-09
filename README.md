# Ragnaros

Petit outil perso pour comprendre pourquoi mon portable rame par moments
(i7-12700H + RTX 3060, Linux).

Ça tourne en fond, ça garde les 2 dernières minutes, et quand ça part en vrille
ça écrit tout dans `incidents/`. Ensuite `report` résume le fichier.

Pas de dépendance, juste Go.

## Utilisation

```sh
make build
./ragnaros watch
./ragnaros report incidents/incident-20240906-212008.jsonl
```

`./ragnaros watch -h` pour toutes les options. Celles que j'utilise :

```sh
./ragnaros watch -threshold 1 -samples 2   # pour tester, ça déclenche direct
./ragnaros watch -interval 500ms
./ragnaros watch -dir /tmp/ragnaros
```

Pour tester l'alerte « un cœur » sans charger la machine, lancer ça dans un
autre terminal pendant que `watch` tourne :

```sh
timeout 10 sh -c 'while :; do :; done'
```

Normalement ça affiche `Alert: sh uses 100 % of a core` et un fichier apparaît.

## Ce qui est mesuré

Chaque seconde : CPU, load, mémoire libre, swap, température CPU (`coretemp`),
vitesse des cœurs, et le top 5 des process (100 % = un cœur).

Le GPU NVIDIA aussi, via `nvidia-smi`, mais seulement s'il est déjà réveillé.
Sinon on le réveille et il chauffe le CPU (même caloduc).

Une alerte part après 5 mesures d'affilée avec :

- CPU total > 90 % (`-threshold`)
- un process > 95 % d'un cœur (`-core-threshold`). Typiquement un jeu : il peut
  ramer alors que le CPU total est à 5 %.
- CPU > 90 °C (`-temp-threshold`)

Un seul fichier par pic. Ctrl+C pendant que ça rame enregistre aussi.

## Exemple

```
$ ./ragnaros watch
Ragnaros is watching. Press Ctrl+C to stop.

17:40:39  CPU   9.6 %  load 1.99  free memory 43.3 GiB  swap 0 MiB  54 °C  speed 1600/4700 MHz  GPU 10 % 53 °C 17 W 43/6144 MiB
    13504    43.1 %   1600/4700 MHz  /opt/google/chrome/chrome --type=renderer --crashpad-hand...
     4141    24.0 %   1204/3500 MHz  /opt/google/chrome/chrome
     2491    13.0 %   1100/3500 MHz  /usr/bin/kwin_wayland --wayland-fd 7 --socket wayland-0 -...
```

Le rapport après le `timeout 8 sh -c 'while :; do :; done'` :

```
$ ./ragnaros report incidents/incident-20240906-183339.jsonl
Incident file: incidents/incident-20240906-183339.jsonl
Period:             2024-09-06 18:33:33 -> 2024-09-06 18:33:39 (6s, 7 samples)
Alert:              sh uses 101 % of a core
CPU peak:           7.0 % at 18:33:39
Load peak:          2.00 at 18:33:33
Lowest free memory: 45.4 GiB at 18:33:36
Most swap used:     none
Hottest CPU:        66 °C at 18:33:37
Lowest CPU speed:   3000 MHz (max 4700) at 18:33:33
GPU peak:           13 % at 18:33:37, up to 52 °C, 16 W, 43/6144 MiB (asleep 0/7)

Busiest processes (SEEN = times in the top 5):
      PID    SEEN   MAX CPU       AVG SPEED  COMMAND
   351250     5/7   100.8 %   2999/4700 MHz  sh -c while :; do :; done
      154     7/7     5.0 %   1100/4600 MHz  irq/9-acpi

100 % = one full core. AVG SPEED = average speed of the core it ran on.
This shows what was running, not proof of what caused the slowdown.
```

## Limites

- Rien avant le lancement de `watch`, évidemment.
- Un truc qui reste à fond tout le temps donne une seule alerte, au début. Pour
  la suite c'est la température qui compte.
- Une compil déclenche l'alerte « un cœur », c'est normal.
- `nvidia-smi` apparaît parfois dans le top : c'est lui qui est lancé chaque
  seconde.
- Les commandes sont coupées à 60 caractères.
- Pas de détail par cœur. Linux balade les threads d'un cœur à l'autre, donc un
  thread à 100 % ressemble à 3 ou 4 cœurs à moitié pleins. Le % par process est
  plus fiable.

## Tests

`make test`. Les données sont inventées, sauf un test qui lit le vrai `/proc`.
