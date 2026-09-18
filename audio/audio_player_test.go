package audio

import (
	"bytes"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"io"
	"testing"
	"time"
)

func TestGeneratedEffectsFinishInSilence(t *testing.T) {
	for _, effect := range []Effect{Alert, KeyClick} {
		s := newAudioSource()
		s.mutex.Lock()
		s.positions[effect] = 0
		s.active[effect] = true
		s.mutex.Unlock()
		buf := make([]byte, (len(s.samples[effect])+100)*soundFrameBytes)
		s.Read(buf[:17])
		s.Read(buf[17:])
		if bytes.Equal(buf, make([]byte, len(buf))) {
			t.Fatalf("effect %d is silent", effect)
		}
		for i := 0; i < len(buf); i += soundFrameBytes {
			if !bytes.Equal(buf[i:i+2], buf[i+2:i+4]) {
				t.Fatalf("effect %d: stereo mismatch at %d", effect, i)
			}
		}
		for _, b := range buf[len(buf)-200:] {
			if b != 0 {
				t.Fatalf("effect %d did not stop", effect)
			}
		}
	}
}

func TestEmbeddedRecordingsDecode(t *testing.T) {
	for name, data := range map[string][]byte{"engine": engineMP3, "gunfire": gunfireMP3, "explosion": explosionMP3} {
		pcm, err := decodeMP3(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(pcm) == 0 || len(pcm)%soundFrameBytes != 0 {
			t.Fatalf("%s: invalid stereo PCM length %d", name, len(pcm))
		}
	}
}

func TestKeyClickIsBrief(t *testing.T) {
	s := newAudioSource()
	if got, want := len(s.samples[KeyClick]), int(DefaultSettings.KeyClickDuration*soundSampleRate); got != want {
		t.Fatalf("click length = %d samples, want %d", got, want)
	}
}

func TestMP3PreservesDecodedSamples(t *testing.T) {
	for name, data := range map[string][]byte{"engine": engineMP3, "gunfire": gunfireMP3, "explosion": explosionMP3} {
		stream, err := mp3.DecodeWithSampleRate(soundSampleRate, bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		want, err := io.ReadAll(stream)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decodeMP3(data)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s: playback altered decoded samples", name)
		}
	}
}

func TestRecordedDurationUsesStereo16BitFrames(t *testing.T) {
	p := &Player{recorded: map[Effect][]byte{Engine: make([]byte, soundSampleRate*soundFrameBytes)}}
	if got := p.duration(Engine); got != time.Second {
		t.Fatalf("duration = %s", got)
	}
}

func TestInvalidEffectsAndMutedAlert(t *testing.T) {
	p := &Player{source: newAudioSource()}
	for _, effect := range []Effect{-1, effectCount, Alert} {
		p.Play(effect)
	}
	buf := make([]byte, 1024)
	p.source.Read(buf)
	if !bytes.Equal(buf, make([]byte, len(buf))) {
		t.Fatal("invalid or muted effect produced audio")
	}
}
