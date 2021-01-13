package main

import (
	"io"
	"sync"

	"github.com/hajimehoshi/oto"
)

// A trivial player generating strictly rectangular waves of given frequency on 4 channels.
type AudioPlayer struct {
	player      *oto.Player
	mutex       sync.Mutex
	frequencies [4]byte
	currentPos  int
	origBuf     []byte
	buf         []byte
}

func (p *AudioPlayer) Read(buf []byte) (int, error) {
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

func NewAudioPlayer(context *oto.Context) *AudioPlayer {
	p := &AudioPlayer{
		player:  context.NewPlayer(),
		origBuf: make([]byte, 4096)}
	go func() {
		for {
			io.Copy(p.player, p)
		}
	}()
	return p
}

func (p *AudioPlayer) SetFrequency(channel int, freq byte) {
	p.frequencies[channel] = freq
}
func (p *AudioPlayer) Close() {
	p.player.Close()
}
