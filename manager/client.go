package manager

import (
	"log"
	"quake2srv/shared"

	"github.com/gorilla/websocket"
)

type clientConnState int

const (
	clientIdle   clientConnState = 0
	clientInGame clientConnState = 1
)

type QWSClient interface {
	Handler()
}

type qWSClient struct {
	conn   *websocket.Conn
	queues GameQueueHandler
	state  clientConnState
	game   QGame
}

func closeHandler(code int, text string) error {
	println("closeHandler", code, text)
	return nil
}

func CreateWSClient(conn *websocket.Conn, q GameQueueHandler) QWSClient {
	ws := &qWSClient{}
	ws.conn = conn
	ws.queues = q
	ws.state = clientIdle
	ws.conn.SetCloseHandler(closeHandler)
	return ws
}

func (cl *qWSClient) Addr() string {
	return cl.conn.RemoteAddr().String()
}

func (cl *qWSClient) Transmit(data []byte) {
	cl.conn.WriteMessage(2, data)
}

func (cl *qWSClient) Handler() {
	for {
		_, message, err := cl.conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			if cl.game != nil {
				cl.game.Disconnect(cl.Addr())
			}
			break
		}
		if cl.state == clientIdle {
			if message[0] != 0xFF || message[1] != 0xFF || message[2] != 0xFF || message[3] != 0xFF {
				log.Println("Received connected message when in idle state")
				continue
			}
			str := string(message[4:])
			println(str)
			args := shared.Cmd_TokenizeString(str, false)
			if len(args) == 0 {
				continue
			}
			switch args[0] {
			case "singleplayer":
				if len(args) != 2 {
					log.Println("Invalid parameters for singleplayer", len(args))
					continue
				}
				status, game := cl.queues.singleQueue.addToQueue(cl, args[1])
				switch status {
				case STATUS_QUEUED:
					cl.conn.WriteMessage(2, []byte("QUEUED"))
					cl.game = nil
				case STATUS_INGAME:
					cl.conn.WriteMessage(2, []byte("GAME"))
					cl.state = clientInGame
					cl.game = game
				case STATUS_ERROR:
					cl.conn.WriteMessage(2, []byte("ERROR"))
					cl.game = nil
				}

			default:
				log.Println("Unknown command", args[0])
			}
		} else if cl.state == clientInGame {
			cl.game.RxHandler(cl.Addr(), message)
		}
	}
}
