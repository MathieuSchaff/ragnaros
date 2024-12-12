# Ragnaros

Mon petit outil pour savoir pourquoi le PC rame.

Il tourne en fond, garde les 2 dernières minutes en mémoire, et quand ça chauffe
il enregistre tout dans un fichier. Ensuite `report` résume ce qui s'est passé.

Fait pour mon portable : Intel i7-12700H + RTX 3060, Linux. Go, sans dépendance.

## Lancer

```sh
go build -o ragnaros .
./ragnaros watch                     # Ctrl+C pour arrêter
./ragnaros report incidents/incident-20240906-212008.jsonl
```

Options utiles de `watch` (toutes : `./ragnaros watch -h`) :

```sh
./ragnaros watch -threshold 1 -samples 2    # déclenche tout de suite, pour tester
./ragnaros watch -interval 500ms            # deux mesures par seconde
./ragnaros watch -dir /tmp/ragnaros         # ailleurs que ./incidents
```

Simuler un programme qui sature un seul cœur (comme un jeu), sans charger la
machine (1 cœur sur 20) :

```sh
./ragnaros watch                                  # terminal 1
timeout 10 sh -c 'while :; do :; done'            # terminal 2
```

Attendu : `Alert: sh uses 100 % of a core`, puis un fichier dans `incidents/`.

## Ce qu'il mesure, chaque seconde

- CPU total, charge, mémoire libre, swap ;
- température du CPU (capteur `coretemp`) ;
- vitesse des cœurs : un cœur chargé loin de son max, c'est souvent la chaleur ;
- GPU NVIDIA (utilisation, température, watts, mémoire) via `nvidia-smi`,
  seulement s'il est déjà réveillé : l'appeler réveillerait la carte, qui
  chaufferait le CPU ;
- les 5 processus qui prennent le plus de CPU (100 % = un cœur entier).

## Quand il enregistre

Après 5 mesures d'affilée où au moins un de ces cas est vrai :

- CPU total au-dessus de 90 % (`-threshold`) ;
- un processus au-dessus de 95 % d'un cœur (`-core-threshold`) : un programme
  qui n'utilise qu'un cœur peut ramer alors que le CPU total reste à 5 % ;
- CPU au-dessus de 90 °C (`-temp-threshold`).

Un seul fichier par pic, avec la raison de l'alerte. Si on fait Ctrl+C pendant
que ça rame, il enregistre aussi.

## Exemple

```
$ ./ragnaros watch
Ragnaros is watching. Press Ctrl+C to stop.

17:40:39  CPU   9.6 %  load 1.99  free memory 43.3 GiB  swap 0 MiB  54 °C  speed 1600/4700 MHz  GPU 10 % 53 °C 17 W 43/6144 MiB
    13504    43.1 %   1600/4700 MHz  /opt/google/chrome/chrome --type=renderer --crashpad-hand...
     4141    24.0 %   1204/3500 MHz  /opt/google/chrome/chrome
     2491    13.0 %   1100/3500 MHz  /usr/bin/kwin_wayland --wayland-fd 7 --socket wayland-0 -...
```

Rapport après `timeout 8 sh -c 'while :; do :; done'` :

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

## À savoir

- Ce qui se passe avant le lancement de `watch` n'est pas connu.
- Un programme qui reste à fond tout le temps ne donne qu'une alerte, au début.
  Pour les lags d'après, c'est l'alerte température qui compte.
- Une compilation déclenche aussi l'alerte « un cœur ».
- `nvidia-smi` est lancé chaque seconde quand le GPU est réveillé : on le voit
  parfois dans le top.
- Les noms trop longs sont coupés à 60 caractères.
- Pas de détail cœur par cœur : Linux déplace un programme d'un cœur à l'autre
  plusieurs fois par seconde, donc un seul thread à 100 % apparaît comme 3 ou 4
  cœurs à moitié occupés. Le % par processus (100 % = un cœur) et la vitesse de
  son cœur sont plus fiables.

## Tests

```sh
go test ./...
```

Données inventées, sauf un test qui lit le vrai `/proc` une fois. Rien ne charge
la machine.
