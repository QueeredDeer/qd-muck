package playerreg

import (
	"github.com/QueeredDeer/qd-muck/internal/player"
)

type RegistryCallback struct {
	PlayerName string
	Callback   chan bool
}

type ActivePlayerRegistry struct {
	AddPlayer    chan *player.Player
	RemovePlayer chan string
	QueryPlayer  chan *RegistryCallback
	Done         chan int

	activePlayers map[string]*player.Player
}

func New() *ActivePlayerRegistry {
	apreg := ActivePlayerRegistry{
		AddPlayer:     make(chan *player.Player),
		RemovePlayer:  make(chan string),
		QueryPlayer:   make(chan *RegistryCallback),
		Done:          make(chan int),
		activePlayers: make(map[string]*player.Player),
	}

	return &apreg
}

func (reg *ActivePlayerRegistry) add(p *player.Player) {
	_, found := reg.activePlayers[p.Name]
	if !found {
		reg.activePlayers[p.Name] = p
	}
}

func (reg *ActivePlayerRegistry) remove(name string) {
	delete(reg.activePlayers, name)
}

func (reg *ActivePlayerRegistry) query(rc *RegistryCallback) {
	_, found := reg.activePlayers[rc.PlayerName]
	rc.Callback <- found
}

func (reg *ActivePlayerRegistry) Launch() {
	for {
		select {
		case ap := <-reg.AddPlayer:
			reg.add(ap)
		case rp := <-reg.RemovePlayer:
			reg.remove(rp)
		case qp := <-reg.QueryPlayer:
			reg.query(qp)
		case <-reg.Done:
			return
		}
	}
}
