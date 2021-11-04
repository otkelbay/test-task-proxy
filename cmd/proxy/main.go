package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"test.task/backend/proxy/cmd/proxy/filterer"

	"github.com/gorilla/websocket"
	proxy "test.task/backend/proxy"
)

var (
	addr                  = flag.String("addr", "localhost:8081", "proxy http service address")
	serverAddr            = flag.String("server_addr", "localhost:8080", "server http service address")
	openOrderAmountLimit  = flag.Int("open_order_amount_limit", 10, "open order amount limit")
	openOrderSumLimit     = flag.Float64("open_order_sum_limit", 1000, "open order sum limit")
	upgrader              = websocket.Upgrader{}
	filter                = filterer.Filterer{}
	clientWsConnectionMap = make(map[uint32]*websocket.Conn)
	requestIDClientIDMap  = make(map[uint32]uint32)
)

var serverWsConnection *websocket.Conn

func main() {
	flag.Parse()
	log.SetFlags(0)

	// initializing the proxy http server
	http.HandleFunc("/connect", connect)
	go func() {
		log.Printf("Waiting for proxy connections on %s/connect", *addr)
		log.Fatal(http.ListenAndServe(*addr, nil))
	}()

	// connecting to the server
	u := url.URL{Scheme: "ws", Host: *serverAddr, Path: "/connect"}
	log.Printf("connecting to %s", u.String())
	var err error
	serverWsConnection, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer serverWsConnection.Close()

	// initializing the filterer
	filter = filterer.Filterer{
		Limits:               make(map[uint32]map[string][]float64),
		OpenOrderSumLimit:    *openOrderSumLimit,
		OpenOrderAmountLimit: *openOrderAmountLimit,
	}

	// waiting and processing answers from the server
	sendResponseFromServerToClient(serverWsConnection)
}

func connect(w http.ResponseWriter, r *http.Request) {
	// retrieving websocket connection by upgrading http connection
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	// waiting for orders from client
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		req := proxy.DecodeOrderRequest(message)
		log.Printf("recv: %v", req)

		_, ok := clientWsConnectionMap[req.ClientID]
		if !ok {
			clientWsConnectionMap[req.ClientID] = c
		}

		requestIDClientIDMap[req.ID] = req.ClientID

		if !filter.Filter(&req) {
			log.Printf("req didn't passed filter: %v", req)
			continue
		}

		err = serverWsConnection.WriteMessage(mt, message)
		if err != nil {
			log.Println("write error:", err)
			continue
		}

		log.Printf("sent to server from proxy: %v", req)
	}
}

func sendResponseFromServerToClient(serverWsConnection *websocket.Conn) {
	for {
		mt, mess, err := serverWsConnection.ReadMessage()
		if err != nil {
			log.Printf("recv error: %+v", err)
			return
		}

		resp := proxy.DecodeOrderResponse(mess)
		m := requestIDClientIDMap
		if _, ok := m[resp.ID]; !ok {
			log.Printf("couldn't get client id of request")
			return
		}

		clientConn, ok := clientWsConnectionMap[requestIDClientIDMap[resp.ID]]
		if !ok {
			log.Printf("couldn't get client connection")
			return
		}

		err = clientConn.WriteMessage(mt, proxy.EncodeOrderResponse(resp))
		if err != nil {
			log.Println("write error:", err)
			return
		}

		log.Printf("sent response to client: %v", resp)
	}
}
