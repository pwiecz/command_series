package audio

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"sync"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

const soundSampleRate = 44100
const soundFrameBytes = 4 // two channels, two bytes per sample

//go:embed assets/engine.mp3
var engineMP3 []byte

//go:embed assets/gunfire.mp3
var gunfireMP3 []byte

//go:embed assets/nuclear.mp3
var explosionMP3 []byte

type Effect int

const (
	Engine Effect = iota
	Explosion
	Shooting
)

// Player plays embedded recordings for movement and combat.
type Player struct {
	context   *oto.Context
	recorded  map[Effect][]byte
	playersMu sync.Mutex
	players   []*oto.Player
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

func NewPlayer(context *oto.Context) (*Player, error) {
	p := &Player{context: context}
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
	return p, nil
}

func (p *Player) Play(effect Effect) {
	if p == nil || p.context == nil {
		return
	}
	pcm, ok := p.recorded[effect]
	if !ok {
		return
	}
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
	player.Play()
	p.players = append(kept, player)
}

func (p *Player) Close() {
	if p == nil {
		return
	}
	p.playersMu.Lock()
	defer p.playersMu.Unlock()
	for _, player := range p.players {
		_ = player.Close()
	}
	p.players = nil
}
