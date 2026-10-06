# Attribution

`engine/gnubg` (+ `math32`, `met`, `sigmoid`, `data/`) is ported from
[foochu/bgweb-api](https://github.com/foochu/bgweb-api) (MIT, (c) 2022 Rami
Keränen — see MIT grant below), which itself ports the evaluator from
[GNU Backgammon](https://www.gnu.org/software/gnubg/) (GPLv3, (c) 1999-2022
Gary Wong and contributors). Weights (`gnubg.weights`) and bearoff databases
(`gnubg_os0.bd`, `gnubg_ts0.bd`) are gnubg data files; match-equity tables in
`data/met/` are Kazaross/Rockwell-Kazaross (shipped upstream with bgweb-api).

Treat this directory as GPL-covered to satisfy both lineages: keep this repo
open source, preserve notices, state changes here.

Local changes vs upstream: `FindMovesEx` threads search depth + eval noise
(`gnubg.go`); import paths retargeted to `fair_backgammon/engine/gnubg`;
`internal/` layout flattened (Go internal rule); tests dropped (upstream
vectors live in `engine/*_test.go` now).

MIT grant (bgweb-api LICENSE):

    Permission is hereby granted, free of charge, to any person obtaining a copy
    of this software and associated documentation files (the "Software"), to deal
    in the Software without restriction, including without limitation the rights
    to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
    copies of the Software, and to permit persons to whom the Software is
    furnished to do so, subject to the following conditions:

    The above copyright notice and this permission notice shall be included in all
    copies or substantial portions of the Software.
