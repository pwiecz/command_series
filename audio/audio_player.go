package audio

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/oto/v2"
)

const soundSampleRate = 44100
const soundFrameBytes = 4 // two channels, two bytes per sample

//go:embed assets/engine.mp3
var engineMP3 []byte

//go:embed assets/gunfire.mp3
var gunfireMP3 []byte

//go:embed assets/nuclear.mp3
var explosionMP3 []byte

// Settings controls the generated alert and keyboard-click effects.
type Settings struct {
	Volume           float64
	AlertDuration    float64
	KeyClickDuration float64
	AlertFrequency   float64
	AlertAmplitude   float64
}

var DefaultSettings = Settings{
	Volume:           1.0,
	AlertDuration:    0.05,
	KeyClickDuration: 0.018,
	AlertFrequency:   550,
	AlertAmplitude:   500,
}

type Effect int

const (
	Engine Effect = iota
	Alert
	Explosion
	Shooting
	KeyClick
	effectCount
)

// Player plays embedded recordings for movement and combat, and synthesizes
// the short alert and keyboard-click sounds.
type Player struct {
	player    oto.Player
	source    *audioSource
	context   *oto.Context
	recorded  map[Effect][]byte
	playersMu sync.Mutex
	players   []oto.Player
	settings  Settings
}

type audioSource struct {
	mutex     sync.Mutex
	samples   [effectCount][]float64
	positions [effectCount]int
	active    [effectCount]bool
	frameByte int
	frame     [soundFrameBytes]byte
	settings  Settings
}

func newAudioSource() *audioSource { return newAudioSourceWithSettings(DefaultSettings) }

func newAudioSourceWithSettings(settings Settings) *audioSource {
	s := &audioSource{settings: settings}
	s.samples[Alert] = synthesizeEffect(Alert, settings)
	s.samples[KeyClick] = synthesizeEffect(KeyClick, settings)
	return s
}

func (p *audioSource) Read(buf []byte) (int, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for i := range buf {
		if p.frameByte == 0 {
			var mixed float64
			for _, effect := range []Effect{Alert, KeyClick} {
				if !p.active[effect] {
					continue
				}
				mixed += p.samples[effect][p.positions[effect]]
				p.positions[effect]++
				if p.positions[effect] == len(p.samples[effect]) {
					p.active[effect] = false
				}
			}
			mixed = math.Max(-1, math.Min(1, mixed)) * p.settings.Volume
			sample := uint16(int16(math.Round(32767 * mixed)))
			binary.LittleEndian.PutUint16(p.frame[:2], sample)
			binary.LittleEndian.PutUint16(p.frame[2:], sample)
		}
		buf[i] = p.frame[p.frameByte]
		p.frameByte = (p.frameByte + 1) % soundFrameBytes
	}
	return len(buf), nil
}

func synthesizeEffect(effect Effect, settings Settings) []float64 {
	var duration float64
	switch effect {
	case Alert:
		duration = settings.AlertDuration
	case KeyClick:
		duration = settings.KeyClickDuration
	default:
		return nil
	}
	samples := make([]float64, int(duration*soundSampleRate))
	rng := rand.New(rand.NewSource(int64(effect) + 1))
	for i := range samples {
		t := float64(i) / soundSampleRate
		noise := rng.Float64()*2 - 1
		var value float64
		switch effect {
		case Alert:
			value = settings.AlertAmplitude * math.Sin(2*math.Pi*settings.AlertFrequency*t) * math.Sin(math.Pi*t/duration)
		case KeyClick:
			value = (0.32*noise + 0.08*math.Sin(2*math.Pi*780*t)) * math.Exp(-220*t)
		}
		fade := math.Min(1, math.Min(t/0.003, (duration-t)/0.025))
		samples[i] = value * fade
	}
	return samples
}

func decodeMP3(data []byte) ([]byte, error) {
	stream, err := mp3.DecodeWithSampleRate(soundSampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	pcm16, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	if len(pcm16)%soundFrameBytes != 0 {
		return nil, fmt.Errorf("decoded MP3 has incomplete stereo frame")
	}
	return pcm16, nil
}

func NewPlayer(context *oto.Context) (*Player, error) { return newPlayer(context, DefaultSettings) }

func newPlayer(context *oto.Context, settings Settings) (*Player, error) {
	p := &Player{context: context, settings: settings}
	if context == nil {
		return p, nil
	}
	engine, err := decodeMP3(engineMP3)
	if err != nil {
		return nil, fmt.Errorf("decode engine sound: %w", err)
	}
	gunfire, err := decodeMP3(gunfireMP3)
	if err != nil {
		return nil, fmt.Errorf("decode gunfire sound: %w", err)
	}
	explosion, err := decodeMP3(explosionMP3)
	if err != nil {
		return nil, fmt.Errorf("decode explosion sound: %w", err)
	}
	const engineMaxBytes = soundSampleRate * soundFrameBytes // one second of 16-bit stereo PCM
	if len(engine) > engineMaxBytes {
		engine = engine[:engineMaxBytes]
	}
	p.recorded = map[Effect][]byte{Engine: engine, Shooting: gunfire, Explosion: explosion}
	p.source = newAudioSourceWithSettings(settings)
	p.player = context.NewPlayer(p.source)
	if buffered, ok := p.player.(oto.BufferSizeSetter); ok {
		buffered.SetBufferSize(soundSampleRate * soundFrameBytes / 20)
	}
	p.player.Play()
	return p, nil
}

var soundNames = map[string]Effect{
	"engine": Engine, "alert": Alert, "explosion": Explosion,
	"gunfire": Shooting, "shooting": Shooting, "click": KeyClick,
}

func PreviewSound(name string, repeats int, gap time.Duration) error {
	return PreviewSoundWithSettings(name, repeats, gap, DefaultSettings)
}

// PreviewSoundWithSettings plays exactly the same recordings and generated
// effects the game uses.
func PreviewSoundWithSettings(name string, repeats int, gap time.Duration, settings Settings) error {
	if repeats < 1 {
		return fmt.Errorf("repeats must be at least 1")
	}
	if gap < 0 {
		return fmt.Errorf("gap cannot be negative")
	}
	if settings.Volume < 0 || settings.Volume > 1 || settings.AlertDuration <= 0 || settings.AlertFrequency <= 0 || settings.AlertAmplitude < 0 {
		return fmt.Errorf("invalid sound settings")
	}
	var effects []Effect
	if name == "all" {
		effects = []Effect{Engine, Alert, Explosion, Shooting, KeyClick}
	} else {
		effect, ok := soundNames[name]
		if !ok {
			return fmt.Errorf("unknown sound %q (use engine, alert, explosion, gunfire, shooting, click, or all)", name)
		}
		effects = []Effect{effect}
	}
	context, ready, err := oto.NewContext(soundSampleRate, 2, 2)
	if err != nil {
		return fmt.Errorf("create audio context: %w", err)
	}
	<-ready
	player, err := newPlayer(context, settings)
	if err != nil {
		return err
	}
	defer player.Close()
	for _, effect := range effects {
		for i := 0; i < repeats; i++ {
			player.Play(effect)
			time.Sleep(player.duration(effect) + gap)
		}
	}
	return nil
}

func (p *Player) duration(effect Effect) time.Duration {
	if pcm, ok := p.recorded[effect]; ok {
		return time.Duration(float64(len(pcm)) / float64(soundSampleRate*soundFrameBytes) * float64(time.Second))
	}
	if p.source != nil {
		return time.Duration(float64(len(p.source.samples[effect])) / soundSampleRate * float64(time.Second))
	}
	return 0
}

func (p *Player) Play(effect Effect) {
	if p == nil || effect < 0 || effect >= effectCount || effect == Alert {
		return
	}
	if pcm, ok := p.recorded[effect]; ok && p.context != nil {
		p.playersMu.Lock()
		defer p.playersMu.Unlock()
		kept := p.players[:0]
		for _, player := range p.players {
			if player.IsPlaying() {
				kept = append(kept, player)
			} else {
				_ = player.Close()
			}
		}
		player := p.context.NewPlayer(bytes.NewReader(pcm))
		player.SetVolume(p.settings.Volume)
		player.Play()
		p.players = append(kept, player)
		return
	}
	if p.source == nil || len(p.source.samples[effect]) == 0 {
		return
	}
	p.source.mutex.Lock()
	defer p.source.mutex.Unlock()
	p.source.positions[effect] = 0
	p.source.active[effect] = true
}

func (p *Player) Close() {
	if p == nil {
		return
	}
	if p.player != nil {
		_ = p.player.Close()
	}
	p.playersMu.Lock()
	defer p.playersMu.Unlock()
	for _, player := range p.players {
		_ = player.Close()
	}
	p.players = nil
}
