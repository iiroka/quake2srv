package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"quake2srv/manager"
	"quake2srv/shared"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8081", "http service address")
var singleQueues = flag.Int("single", 5, "number of single player games")
var coopQueues = flag.Int("coop", 5, "number of coop games")
var dmQueues = flag.Int("dm", 5, "number of death match games")

var filesystem shared.QFileSystem

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var queueHandler manager.GameQueueHandler

func pong(w http.ResponseWriter, r *http.Request) {
	println("PING")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Write([]byte("pong"))
}

func connect(w http.ResponseWriter, r *http.Request) {
	println("connect")
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	cl := manager.CreateWSClient(c, queueHandler)
	go clientHandler(cl)
}

func qfile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[7:]
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Content-Type", "application/octet-stream")

	bfr, err := filesystem.LoadFile(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else if bfr == nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.Write(bfr)
	}
}

func main() {
	flag.Parse()

	dir, _ := os.UserHomeDir()
	filesystem = shared.InitFilesystem(dir, false)

	queueHandler = *manager.CreateGameQueueHandler(*singleQueues, *coopQueues, *dmQueues, filesystem)

	http.HandleFunc("/ping", pong)
	http.HandleFunc("/connect", connect)
	http.HandleFunc("/qfile/", qfile)
	println("Starting to listen...")
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func clientHandler(cl manager.QWSClient) {
	cl.Handler()
}
