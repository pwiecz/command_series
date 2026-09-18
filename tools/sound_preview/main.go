// Command sound_preview plays the exact in-game effects without launching the game.
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/pwiecz/command_series/audio"
)

func main() {
	sound := flag.String("sound", "all", "sound to play: engine, alert, explosion, gunfire, click, or all")
	repeat := flag.Int("repeat", 1, "times to play each selected sound")
	gap := flag.Duration("gap", 180*time.Millisecond, "silence between repeated sounds")
	settings := audio.DefaultSettings
	volume := flag.Float64("volume", settings.Volume, "overall output level (0 to 1)")
	frequency := flag.Float64("frequency", settings.AlertFrequency, "alert tone frequency in Hz")
	duration := flag.Duration("duration", time.Duration(settings.AlertDuration*float64(time.Second)), "alert duration")
	amplitude := flag.Float64("amplitude", settings.AlertAmplitude, "alert waveform amplitude")
	flag.Parse()
	settings.Volume = *volume
	settings.AlertFrequency = *frequency
	settings.AlertDuration = duration.Seconds()
	settings.AlertAmplitude = *amplitude
	fmt.Printf("Playing %s (repeat=%d, volume=%g)\n", *sound, *repeat, settings.Volume)
	if *sound == "alert" || *sound == "all" {
		fmt.Printf("Alert: frequency=%g Hz, duration=%s, amplitude=%g\n", settings.AlertFrequency, *duration, settings.AlertAmplitude)
	}
	if err := audio.PreviewSoundWithSettings(*sound, *repeat, *gap, settings); err != nil {
		log.Fatal(err)
	}
}
