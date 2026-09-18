# Command Series engine

An engine for playing [Command Series](https://www.mobygames.com/game-group/microprose-command-series-games) games ([Crusade in Europe](https://www.mobygames.com/game/crusade-in-europe/), [Decision in the Desert](https://www.mobygames.com/game/decision-in-the-desert/), [Conflict in Vietnam](https://www.mobygames.com/game/conflict-in-vietnam/)) developed by Sid Meier in the mid-eighties and published by MicroProse.

# Using

Obtain an ATR image of Atari version of one of the games and run `$ command_series <diskimage.atr>`.

## Audio

Adds sound effects to the original engine:

- Unit movement: `audio/assets/engine.mp3`.
- Normal attacks: `audio/assets/gunfire.mp3`.
- Long-range attacks (air and artillery strikes): `audio/assets/nuclear.mp3`.

Preview the sounds without starting a game:

```sh
go run ./tools/sound_preview -sound engine
go run ./tools/sound_preview -sound gunfire
go run ./tools/sound_preview -sound explosion
```

The MP3s are embedded in the executable. To replace a sound, update its file
in `audio/assets/` and rebuild:

```sh
go build -o command_series ./cmd
```

# Missing features

* Bug fixes ~~, many bug-fixes~~
* ~~Save/load~~
* Music
* Intro and ending
* Color cycling of cursor/icons, etc.
