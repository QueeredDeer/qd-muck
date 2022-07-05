package playerreg

import (
	"testing"

	"github.com/QueeredDeer/qd-muck/internal/player"
)

func TestQueryEmptyPlayerRegistry(t *testing.T) {
	reg := New()
	go reg.Launch()

	cb := RegistryCallback{
		PlayerName: "Charlie",
		Callback:   make(chan bool),
	}

	reg.QueryPlayer <- &cb

	resp := <-cb.Callback
	if resp {
		// found un-added player
		t.Errorf("empty registry reported player present")
	}

	reg.Done <- 1
}

func TestAddPlayerRegistry(t *testing.T) {
	reg := New()
	go reg.Launch()

	name := "Charlie"

	tplayer := player.New(name, "")
	reg.AddPlayer <- tplayer

	cb := RegistryCallback{
		PlayerName: name,
		Callback:   make(chan bool),
	}

	reg.QueryPlayer <- &cb

	resp := <-cb.Callback
	if !resp {
		// didn't find added player
		t.Errorf("could not find expected player '" + name + "'")
	}

	reg.Done <- 1
}

func TestQueryNameVariety(t *testing.T) {
	reg := New()
	go reg.Launch()

	name := "aB23 _#$-vL;="

	tplayer := player.New(name, "")
	reg.AddPlayer <- tplayer

	cb := RegistryCallback{
		PlayerName: name,
		Callback:   make(chan bool),
	}

	reg.QueryPlayer <- &cb

	resp := <-cb.Callback
	if !resp {
		// didn't find added player
		t.Errorf("could not find expected player '" + name + "'")
	}

	reg.Done <- 1
}

func TestAddExistingPlayer(t *testing.T) {
	reg := New()
	go reg.Launch()

	name := "Charlie"

	tplayer := player.New(name, "")
	reg.AddPlayer <- tplayer

	// duplicate addition
	reg.AddPlayer <- tplayer

	cb := RegistryCallback{
		PlayerName: name,
		Callback:   make(chan bool),
	}

	reg.QueryPlayer <- &cb

	resp := <-cb.Callback
	if !resp {
		// didn't find added player
		t.Errorf("could not find expected player '" + name + "'")
	}

	reg.Done <- 1
}

func TestRemovePlayerRegistry(t *testing.T) {
	reg := New()
	go reg.Launch()

	name := "Charlie"

	tplayer := player.New(name, "")
	reg.AddPlayer <- tplayer

	reg.RemovePlayer <- tplayer.Name

	cb := RegistryCallback{
		PlayerName: name,
		Callback:   make(chan bool),
	}

	reg.QueryPlayer <- &cb

	resp := <-cb.Callback
	if resp {
		t.Errorf("found removed player still in registry")
	}

	reg.Done <- 1
}

func TestRemoveNonexistentPlayer(t *testing.T) {
	reg := New()
	go reg.Launch()

	name := "Charlie"

	reg.RemovePlayer <- name

	cb := RegistryCallback{
		PlayerName: name,
		Callback:   make(chan bool),
	}

	reg.QueryPlayer <- &cb

	resp := <-cb.Callback
	if resp {
		t.Errorf("found removed player still in registry")
	}

	reg.Done <- 1
}
