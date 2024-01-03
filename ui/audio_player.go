package ui

import (
	"sync"

	"github.com/ebitengine/oto/v3"
)

// A trivial player generating strictly rectangular waves of given frequency on 4 channels.
type AudioPlayer struct {
	player *oto.Player
	source *audioSource
}

type audioSource struct {
	mutex       sync.Mutex
	currentPos  int
	frequencies [4]byte
	origBuf     []byte
	buf         []byte
}

func (p *audioSource) Read(buf []byte) (int, error) {
	if len(p.buf) == 0 {
		p.buf = p.origBuf
		p.mutex.Lock()
		freq := p.frequencies
		p.mutex.Unlock()
		for i := 0; i < len(p.buf); i++ {
			channel := (p.currentPos + i) % 4
			if freq[channel] == 0 {
				p.buf[i] = 128
			} else {
				channelPos := (p.currentPos + i) / 4
				channelLength := 44100 / int(freq[channel])
				if channelPos%channelLength < channelLength/2 {
					p.buf[i] = 0
				} else {
					p.buf[i] = 255
				}
			}
		}
		p.currentPos += len(p.buf)
	}
	n := copy(buf, p.buf)
	p.buf = p.buf[n:]
	return n, nil
}

func (p *audioSource) SetFrequency(channel int, freq byte) {
	p.frequencies[channel] = freq
}

func NewAudioPlayer(context *oto.Context) *AudioPlayer {
	if context == nil {
		return &AudioPlayer{}
	}
	s := &audioSource{
		origBuf: make([]byte, 4096),
	}
	p := &AudioPlayer{
		source: s,
		player: context.NewPlayer(s),
	}
	p.player.Play()
	return p
}

func (p *AudioPlayer) SetFrequency(channel int, freq byte) {
	p.source.SetFrequency(channel, freq)
}

func (p *AudioPlayer) Close() {
	if p.player == nil {
		return
	}
	p.player.Reset()
}
