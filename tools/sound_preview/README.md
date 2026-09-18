# Sound preview

Build and run the preview:

```sh
go build -o sound_preview ./tools/sound_preview
./sound_preview -sound engine
./sound_preview -sound gunfire
./sound_preview -sound explosion
```

`engine`, `gunfire` (also accepted as `shooting`), and `explosion` play the same embedded MP3 files as the game. The source assets are in `audio/assets/engine.mp3`, `audio/assets/gunfire.mp3`, and `audio/assets/nuclear.mp3`.

Engine playback is capped at one second; explosion plays its full recording. Both use 16-bit stereo. Alert playback is disabled.
Click is available in the preview but disabled in the game.

After replacing an MP3, rebuild both executables:

```sh
go build -o command_series ./cmd
go build -o sound_preview ./tools/sound_preview
```
