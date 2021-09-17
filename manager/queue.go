package manager

import (
	"fmt"
	"log"
	"quake2srv/common"
	"quake2srv/server"
	"quake2srv/shared"
	"sync"
)

type GameQueueHandler struct {
	singleQueue     GameQueue
	coopQueue       GameQueue
	deathMatchQueue GameQueue
}

func CreateGameQueueHandler(singleCount, coopCount, dmCount int, fs shared.QFileSystem) *GameQueueHandler {
	q := &GameQueueHandler{}
	q.singleQueue = createGameQueue(singleCount, 1, []string{"+set", "deathmatch", "0", "+set", "coop", "0", "+newgame"}, fs, true)
	q.coopQueue = createGameQueue(coopCount, 8, []string{"+dedicated_start"}, fs, false)
	q.deathMatchQueue = createGameQueue(dmCount, 8, []string{"+dedicated_start"}, fs, false)
	return q
}

// PUBLIC API

type QueueStatus int

const (
	STATUS_ERROR  = 0
	STATUS_QUEUED = 1
	STATUS_INGAME = 2
)

type GameQueueClient interface {
	Addr() string
	Transmit(data []byte)
}

type QGame interface {
	RxHandler(from string, data []byte)
	Disconnect(adr string)
}

type GameQueue interface {
	addToQueue(cl GameQueueClient, skillLevel string) (QueueStatus, QGame)
}

// IMPLEMENTATIONS

type qGame struct {
	players []GameQueueClient
	common  shared.QCommon
	srvr    shared.QServer
	mu      sync.Mutex
}

func (G *qGame) RxHandler(from string, data []byte) {
	G.common.RxHandler(from, data)
}

// Player has been disconnected. Remove from the game
func (G *qGame) Disconnect(adr string) {
	println("Disconnect from", adr)

	// Remove the disconnected player
	G.mu.Lock()
	index := -1
	for i, g := range G.players {
		if g.Addr() == adr {
			index = i
			break
		}
	}
	if index == 0 {
		if len(G.players) == 0 {
			G.players = make([]GameQueueClient, 0)
		} else {
			G.players = G.players[1:]
		}
	} else if index == len(G.players)-1 {
		G.players = G.players[:index]
	} else {
		G.players = append(G.players[:index], G.players[index+1:]...)
	}
	println("PLayers left", len(G.players), index)
	G.mu.Unlock()
	G.common.DisconnectHandler(adr)
}

// QUEUE

type gameQueue struct {
	maxGames      int
	maxPlayers    int
	games         []*qGame
	fillingGame   *qGame
	queued        []GameQueueClient
	params        []string
	fs            shared.QFileSystem
	useSkillLevel bool
	mu            sync.Mutex
}

func createGameQueue(gCount, pCount int, params []string, fs shared.QFileSystem, sl bool) GameQueue {
	q := &gameQueue{}
	q.maxGames = gCount
	q.maxPlayers = pCount
	q.games = make([]*qGame, 0)
	q.fillingGame = nil
	q.queued = make([]GameQueueClient, 0)
	q.params = params
	q.fs = fs
	q.useSkillLevel = sl
	return q
}

func (q *gameQueue) addToQueue(cl GameQueueClient, skillLevel string) (QueueStatus, QGame) {
	println("addToQueue", len(q.queued), len(q.games))
	q.mu.Lock()
	if len(q.queued) > 0 {
		q.queued = append(q.queued, cl)
		q.mu.Unlock()
		return STATUS_QUEUED, nil
	}
	// if q.fillingGame != nil {
	// 	q.fillingGame.players = append(q.fillingGame.players, cl)
	// 	if len(q.fillingGame.players) > q.maxPlayers {

	// 	}
	// }
	if len(q.games) >= q.maxGames {
		q.queued = append(q.queued, cl)
		q.mu.Unlock()
		return STATUS_QUEUED, nil
	}
	g := &qGame{}
	g.players = make([]GameQueueClient, 1)
	g.players[0] = cl
	g.common = common.CreateQuekeCommon(q.fs)
	g.srvr = server.CreateQServer(g.common)
	g.common.SetServer(g.srvr)
	q.games = append(q.games, g)
	q.mu.Unlock()
	params := q.params
	params = append(params, "+set")
	params = append(params, "maxclients")
	params = append(params, fmt.Sprintf("%v", q.maxPlayers))
	if q.useSkillLevel {
		params = append(params, "+set")
		params = append(params, "skill")
		params = append(params, skillLevel)
	}
	g.common.Init(params)
	g.common.RegisterClient(cl.Addr(), txHandler, cl)
	go runGame(g, q)
	return STATUS_INGAME, g
}

func txHandler(data []byte, a interface{}) {
	cl := a.(GameQueueClient)
	cl.Transmit(data)
}

func runGame(G *qGame, q *gameQueue) {
	G.common.MainLoop()
	log.Println("GAME EXIT")
	q.mu.Lock()
	index := -1
	for i, g := range q.games {
		if g == G {
			index = i
			break
		}
	}
	if index < 0 {
		log.Panicln("Cannot find game from queue")
	}
	if index == 0 {
		if len(q.games) == 0 {
			q.games = make([]*qGame, 0)
		} else {
			q.games = q.games[1:]
		}
	} else if index == len(q.games)-1 {
		q.games = q.games[:index]
	} else {
		q.games = append(q.games[:index], q.games[index+1:]...)
	}
	q.mu.Unlock()

}
