package rocket

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"RocketAssist/config"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type RocketCon struct {
	UserId        string
	UserName      string `yaml:"user"`
	DisplayName   string `yaml:"-"`
	Password      string `yaml:"password"`
	AuthToken     string `yaml:"authtoken"`
	HostName      string `yaml:"domain"`
	HostSSL       bool   `yaml:"ssl"`
	HostPort      uint16 `yaml:"port"`
	session       string
	channels      map[string]string
	send          chan interface{}
	receive       chan interface{}
	results       map[string]chan map[string]interface{}
	resultsMutex  sync.RWMutex
	resultsAppend chan struct {
		string  string
		channel chan map[string]interface{}
	}
	resultsDel  chan string
	nextId      chan string
	messages    chan Message
	newMessages chan Message
	quit        chan struct{}
}

const STATUS_ONLINE string = "online"
const STATUS_BUSY string = "busy"
const STATUS_AWAY string = "away"
const STATUS_OFFLINE string = "offline"

func NewConnection(domain string, username string, password string) (*RocketCon, error) {
	log.WithField("message", "Method").Debug("NewConnection")
	var rock RocketCon
	rock.HostName = domain
	rock.UserName = username
	rock.Password = password
	rock.init()
	return &rock, nil
}

func NewConnectionAuthToken(domain string, authtoken string) (*RocketCon, error) {
	log.WithField("message", "Method").Debug("NewConnectionAuth")
	var rock RocketCon
	rock.HostName = domain
	rock.AuthToken = authtoken
	rock.init()
	return &rock, nil
}

func NewConnectionFromConfig(config *config.Config) (*RocketCon, error) {
	var rock RocketCon
	log.WithField("message", "Method").Debug("NewConnectionConfig")
	rock.UserId = config.RocketChat.UserId
	rock.UserName = config.RocketChat.User
	rock.Password = config.RocketChat.Password
	rock.HostName = config.RocketChat.HostName
	rock.HostPort = config.RocketChat.Port
	rock.HostSSL = config.RocketChat.SSL
	rock.AuthToken = config.RocketChat.AuthToken

	if rock.HostName == "" {
		return &rock, errors.New("HostName not set")
	}
	if rock.AuthToken == "" && (rock.UserName == "" || rock.Password == "") {
		return &rock, errors.New("AuthToken not set")
	}

	if rock.HostPort == 0 {
		if rock.HostSSL {
			rock.HostPort = 443
		} else {
			rock.HostPort = 80
		}
	}

	err := rock.init()
	return &rock, err
}

func (rock *RocketCon) init() error {
	log.WithField("message", "INit").Debug("I am in init")
	// Init variables
	rock.send = make(chan interface{}, 1024)
	rock.receive = make(chan interface{}, 1024)
	rock.resultsAppend = make(chan struct {
		string  string
		channel chan map[string]interface{}
	}, 0)
	rock.resultsDel = make(chan string, 1024)
	rock.results = make(map[string]chan map[string]interface{})
	rock.nextId = make(chan string, 0)
	rock.messages = make(chan Message, 1024)
	rock.newMessages = make(chan Message, 1024)
	rock.quit = make(chan struct{}, 0)
	rock.channels = make(map[string]string)

	go rock.run()

	log.WithField("message", "afterRun").Debug("hello")
	// Send Init Messages
	rock.connect()
	err := rock.login()
	if err != nil {
		close(rock.quit)
		return err
	}

	if rock.UserName == "" {
		rock.UserName = rock.RequestUserName(rock.UserId)
	}

	rock.subscribeRooms()
	rock.DisplayName, _ = rock.RequestDisplayName(rock.UserId)
	return nil
}

func (rock *RocketCon) run() {
	log.WithField("message", "Method").Debug("run")

	// Set some websocket tunables
	const socketreadsizelimit = 65536
	const pingtime = 120 * time.Second
	const timeout = 125 * time.Second
	log.WithField("message", "Method").Debug("rock.getWsUrl")
	// Define Websocket URL
	wsURL := rock.getWsURL()

	// Init websocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.WithError(err).WithField("wsURL", wsURL).Error("Cannot initiate websocket")
		close(rock.quit)
	}
	log.WithField("message", "Method").Debug("ws.close")
	defer ws.Close()

	// Configure Websocket using Tunables
	ws.SetReadLimit(socketreadsizelimit)
	ws.SetReadDeadline(time.Now().Add(timeout))
	log.WithField("message", "Method").Debug("BeforePong")
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(timeout))
		return nil
	})

	tick := time.NewTicker(pingtime)
	defer tick.Stop()
	log.WithField("message", "Method").Debug("Tickstop")

	// Manage Method/Subscription Ids
	go func() {
		for i := uint64(0); ; i++ {
			i++
			rock.nextId <- fmt.Sprintf("%d", i)
		}
	}()
	log.WithField("message", "Method").Debug("Subscription")

	// Manage Results map
	go func() {
		for {
			select {
			case addition := <-rock.resultsAppend:
				rock.resultsMutex.Lock()
				rock.results[addition.string] = addition.channel
				rock.resultsMutex.Unlock()
			case remove := <-rock.resultsDel:
				rock.resultsMutex.Lock()
				delete(rock.results, remove)
				rock.resultsMutex.Unlock()
			}
		}
	}()
	log.WithField("message", "Method").Debug("ManageResult MAp")

	// Send Thread
	go func() {
		for {
			msg := <-rock.send
			packet, err := json.Marshal(msg)
			err = ws.WriteMessage(websocket.TextMessage, packet)
			if err != nil {
				log.WithError(err).WithField("packet", packet).Error("Cannot write to websocket.")
				return
			}
		}
	}()
	log.WithField("message", "Method").Debug("1")

	// Read Thread
	for {
		_, raw, err := ws.ReadMessage()
		ws.SetReadDeadline(time.Now().Add(timeout))

		if err != nil {
			log.WithError(err).WithField("ws", ws).Warn("Cannot read websocket.")
			break
		}

		var pack map[string]interface{}
		err = json.Unmarshal(raw, &pack)
		if err != nil {
			log.WithError(err).WithField("raw", raw).Warn("Cannot unmarshal data read from websocket.")
			continue
		}
		log.WithField("message", "ReadMessage").Debug(pack["msg"])

		if msg, ok := pack["msg"]; ok {
			switch msg {
			case "connected":
				if session, ok := pack["session"].(string); ok {
					rock.session = session
				} else {
					log.Warn("Session is nil or not a string")
				}
				log.WithField("message", "Method").Debug("4")
			case "result":
				rock.resultsMutex.RLock()
				if channel, ok := rock.results[pack["id"].(string)]; ok {
					// We want to unlock the resultsMutex before the following blocking operation.
					rock.resultsMutex.RUnlock()
					channel <- pack
					log.WithField("message", "Method").Debug("5")
				} else {
					rock.resultsMutex.RUnlock()
					log.WithField("message", "Method").Debug("6")
				}
				log.WithField("message", "Method").Debug("7")
				rock.resultsDel <- pack["id"].(string)
				log.WithField("message", "Method").Debug("8")
			case "added":
				log.WithField("message", "Method").Debug("9")
				switch pack["collection"].(string) {
				case "users":
					break
				default:
					log.WithField("pack", pack).Trace("Ignored incoming added msg.")
				}
			case "updated":
				break
			case "changed":
				log.WithField("message", "Method").Debug("11")

				// Check if it exists and is not nil

				obj := pack["fields"].(map[string]interface{})["args"].([]interface{})
				log.WithField("message", "Method").Debug("12")

				switch pack["collection"].(string) {
				case "stream-notify-user":
					log.WithField("message", "Method").Debug("13")
					switch obj[0].(string) {
					case "inserted":

						log.WithField("message", "Method").Debug("14")
						log.WithField("message", "Method").Debug(obj[1])
						id := obj[1].(map[string]interface{})["rid"].(string)
						name := obj[1].(map[string]interface{})["fname"].(string)
						log.WithField("message", "Method").Debug("Ok? here")
						rock.channels[id] = name
						rock.subscribeRoom(id)
						log.WithField("message", "Method").Debug("After subsription")
					}
				case "stream-room-messages":
					for _, val := range obj {
						log.WithField("message", "Method").Debug("15")
						message := rock.handleMessageObject(val.(map[string]interface{}))
						log.WithField("message", "Method").Debug("16")
						log.WithField("Method", "message").Debug(message)
						if message.IsNew {
							log.WithField("message", "Stream").Debug("Potential trouble")
							select {
							case rock.newMessages <- message:
								break
							default:
							}
						} else {
							select {
							case rock.messages <- message:
								break
							default:
							}
						}
					}
				}
			case "ready":
				break
			case "ping":
				pong := map[string]string{
					"msg": "pong",
				}
				rock.send <- pong
			default:
				log.WithField("raw", string(raw)).Trace("Ping.")
			}
		}
	}
	close(rock.quit)
	time.Sleep(100 * time.Millisecond) // Ugly hack to make sure rock.quit is closed before we return
}

func (rock *RocketCon) generateId() string {
	return <-rock.nextId
}

func (rock *RocketCon) watchResults(str string) chan map[string]interface{} {
	c := make(chan map[string]interface{})
	rock.resultsAppend <- struct {
		string  string
		channel chan map[string]interface{}
	}{string: str, channel: c}
	return c
}

func (rock *RocketCon) subscribeRoom(rid string) {
	subscribeRoom := map[string]interface{}{
		"msg":  "sub",
		"id":   rock.generateId(),
		"name": "stream-room-messages",
		"params": []interface{}{
			rid,
			false,
		},
	}
	rock.send <- subscribeRoom
}

func (rock *RocketCon) subscribeRooms() error {
	if rock.UserId == "" {
		return errors.New("error: Can't subscribe to rooms if user is not known")
	}
	subscriptionMonitor := map[string]interface{}{
		"msg":  "sub",
		"id":   rock.generateId(),
		"name": "stream-notify-user",
		"params": []interface{}{
			rock.UserId + "/subscriptions-changed",
			false,
		},
	}
	rock.send <- subscriptionMonitor

	subscriptionsGet := map[string]interface{}{
		"method": "subscriptions/get",
		"params": []map[string]interface{}{
			map[string]interface{}{
				"$date": time.Now().Unix(),
			},
		},
	}
	reply, err := rock.runMethod(subscriptionsGet)
	if err != nil {
		return err
	}

	objects := reply["result"].(map[string]interface{})["update"].([]interface{})

	for index, _ := range objects {
		rock.subscribeRoom(objects[index].(map[string]interface{})["rid"].(string))
		if _, ok := objects[index].(map[string]interface{})["name"]; ok {
			name := objects[index].(map[string]interface{})["name"].(string)
			id := objects[index].(map[string]interface{})["rid"].(string)
			rock.channels[id] = name
		}
	}
	return nil
}

func (rock *RocketCon) getHttpURL() string {
	var httpURL string
	if rock.HostSSL {
		httpURL = fmt.Sprintf("https://%s:%d", rock.HostName, rock.HostPort)
	} else {
		httpURL = fmt.Sprintf("http://%s:%d", rock.HostName, rock.HostPort)
	}
	return httpURL
}
func (rock *RocketCon) getWsURL() string {
	httpURL := rock.getHttpURL()
	return strings.Replace(httpURL, "http", "ws", 1) + "/websocket"
}

func (rock *RocketCon) restRequest(str string) []byte {
	// Define Websocket URL
	httpURL := rock.getHttpURL() + str

	// Build Request
	client := &http.Client{}
	request, err := http.NewRequest("GET", httpURL, nil)
	if err != nil {
		log.WithError(err).WithField("httpURL", httpURL).Error("Cannot create new GET request to rocketChat.")
		return make([]byte, 0)
	}
	request.Header.Set("X-Auth-Token", rock.AuthToken)
	request.Header.Set("X-User-Id", rock.UserId)

	// Get Request
	response, err := client.Do(request)
	if err != nil {
		log.WithError(err).WithField("request", request).Error("Cannot perform GET request to rocketChat.")
		return make([]byte, 0)
	}

	// Parse Request
	//defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return body
}

func (rock *RocketCon) runMethod(i map[string]interface{}) (map[string]interface{}, error) {
	id := rock.generateId()
	i["msg"] = "method"
	i["id"] = id
	c := rock.watchResults(id)
	defer close(c)
	rock.send <- i
	reply := <-c
	if _, ok := reply["error"]; ok {
		if _, ok := reply["error"].(map[string]interface{})["error"]; ok {
			//errNo := reply["error"].(map[string]interface{})["error"].(string)
			errNo := reply["error"].(map[string]interface{})["error"].(float64)
			errType := reply["error"].(map[string]interface{})["errorType"].(string)
			return reply, fmt.Errorf("Rocket.Chat replied with an error: %f %s", errNo, errType)
		} else {
			return reply, errors.New("Unknown error")
		}
	}
	return reply, nil
}

func (rock *RocketCon) connect() {
	init := map[string]interface{}{
		"msg":     "connect",
		"version": "1",
		"support": []string{"1", "pre2", "pre1"},
	}
	rock.send <- init
}

func (rock *RocketCon) login() error {
	var obj map[string]interface{}
	if rock.AuthToken == "" {
		passhash := fmt.Sprintf("%x", sha256.Sum256([]byte(rock.Password)))
		obj = map[string]interface{}{
			"method": "login",
			"params": []map[string]interface{}{
				map[string]interface{}{
					"user": map[string]interface{}{
						"username": rock.UserName,
					},
					"password": map[string]interface{}{
						"digest":    passhash,
						"algorithm": "sha-256",
					},
				},
			},
		}
	} else {
		obj = map[string]interface{}{
			"method": "login",
			"params": []map[string]interface{}{
				map[string]interface{}{
					"resume": rock.AuthToken,
				},
			},
		}
	}

	reply, err := rock.runMethod(obj)
	if err != nil {
		return err
	}
	rock.UserId = reply["result"].(map[string]interface{})["id"].(string)
	rock.AuthToken = reply["result"].(map[string]interface{})["token"].(string)
	return nil
}

func (rock *RocketCon) GetMessage() (Message, error) {
	var msg Message
	select {
	case msg := <-rock.messages:
		return msg, nil
	case msg := <-rock.newMessages:
		return msg, nil
	case <-rock.quit:
		return msg, errors.New("The rocket connection has been closed")
	}
}

func (rock *RocketCon) GetNewMessage() (Message, error) {
	var msg Message
	log.WithField("Method", "GetNewMessage").Debug(rock.newMessages)
	log.WithField("Method", "GetNewMessageQUIT").Debug(rock.quit)
	select {
	case msg := <-rock.newMessages:
		return msg, nil
	case <-rock.quit:
		return msg, errors.New("The rocket connection has been closed")
	}
}

func (rock *RocketCon) RequestUserName(userid string) string {
	res := rock.restRequest("/api/v1/users.info?userId=" + userid)
	var m map[string]interface{}
	err := json.Unmarshal(res, &m)
	if err != nil {
		log.WithError(err).WithField("response", res).Error("Error while parsing userinfo response.")
		return ""
	}
	return m["user"].(map[string]interface{})["name"].(string)
}

func (rock *RocketCon) RefreshChannelCache() error {
	obj := map[string]interface{}{
		"method": "rooms/get",
	}

	reply, err := rock.runMethod(obj)
	if err != nil {
		return err
	}
	for _, val := range reply["result"].([]interface{}) {
		if _, ok := val.(map[string]interface{})["fname"]; ok {
			name := val.(map[string]interface{})["fname"].(string)
			id := val.(map[string]interface{})["_id"].(string)
			rock.channels[id] = name
		}
	}
	return err
}

func (rock *RocketCon) requestMessageObj(mid string) map[string]interface{} {
	resp := rock.restRequest("/api/v1/chat.getMessage?msgId=" + mid)
	var m map[string]interface{}
	err := json.Unmarshal(resp, &m)
	if err != nil {
		log.WithError(err).WithField("response", resp).Error("Error while parsing chat.getMessage response.")
		return m
	}
	return m
}

func (rock *RocketCon) RequestDisplayName(uid string) (string, error) {
	resp := rock.restRequest("/api/v1/users.info?userId=" + uid)
	var m map[string]interface{}
	err := json.Unmarshal(resp, &m)
	if err != nil {
		return "", err
	}
	if user, ok := m["user"]; ok {
		if name, ok := user.(map[string]interface{})["name"]; ok {
			return name.(string), nil
		}
	}
	return "", errors.New("Some error")
}

func (rock *RocketCon) RequestMessage(mid string) (Message, error) {
	var msg Message
	obj := rock.requestMessageObj(mid)
	if _, ok := obj["message"]; ok {
		msg = rock.handleMessageObject(obj["message"].(map[string]interface{}))
		return msg, nil
	}
	return msg, errors.New("Some error")
}

func (rock *RocketCon) SendMessage(rid string, text string) (Message, error) {
	obj := map[string]interface{}{
		"method": "sendMessage",
		"params": []map[string]interface{}{
			map[string]interface{}{
				"rid": rid,
				"msg": text,
			},
		},
	}

	var msg Message
	reply, err := rock.runMethod(obj)
	if err != nil {
		return msg, err
	}
	if replyObj, ok := reply["result"].(map[string]interface{}); ok {
		msg = rock.handleMessageObject(replyObj)
	}
	msg.IsMe = true
	return msg, nil
}

func (rock *RocketCon) DM(username string, text string) (Message, error) {
	obj := map[string]interface{}{
		"method": "createDirectMessage",
		"params": []string{
			username,
		},
	}

	reply, err := rock.runMethod(obj)
	if err != nil {
		return Message{}, err
	}
	rid := reply["result"].(map[string]interface{})["rid"].(string)
	return rock.SendMessage(rid, text)
}

func (rock *RocketCon) React(mid string, emoji string) error {
	reaction := map[string]interface{}{
		"method": "setReaction",
		"params": []string{
			emoji,
			mid,
		},
	}

	_, err := rock.runMethod(reaction)
	return err
}

func (rock *RocketCon) UserDefaultStatus(status string) error {
	reaction := map[string]interface{}{
		"method": "UserPresence:setDefaultStatus",
		"params": []string{
			status,
		},
	}

	_, err := rock.runMethod(reaction)
	return err
}

func (rock *RocketCon) UserTemporaryStatus(status string) error {
	reaction := map[string]interface{}{
		"method": "UserPresence:" + status,
		"params": []string{},
	}

	_, err := rock.runMethod(reaction)
	return err
}

func (rock *RocketCon) ListCustomEmojis() ([]string, error) {
	emojis := make([]string, 0)

	reply := rock.restRequest("/api/v1/emoji-custom.list")
	var m map[string]interface{}
	err := json.Unmarshal(reply, &m)
	if err != nil {
		return emojis, err
	}

	if _, ok := m["emojis"]; ok {
		for _, val := range m["emojis"].(map[string]interface{})["update"].([]interface{}) {
			emojis = append(emojis, fmt.Sprintf(":%s:", val.(map[string]interface{})["name"].(string)))
		}
	}
	return emojis, nil
}

func (rock *RocketCon) ListUsersInRoomId(roomId string) ([]string, error) {
	users := make([]string, 0)

	reply := rock.restRequest(fmt.Sprintf("/api/v1/channels.members?roomId=%s&count=1000", roomId))
	var m map[string]interface{}
	err := json.Unmarshal(reply, &m)
	if err != nil {
		return users, err
	}

	if members, ok := m["members"]; ok {
		for _, member := range members.([]interface{}) {
			if username, ok := member.(map[string]interface{})["username"]; ok {
				users = append(users, username.(string))
			}
		}
		return users, nil
	}
	return users, errors.New("Failed to handle members")
}

func (rock *RocketCon) ListUsersInRoom(room string) ([]string, error) {
	roomId := ""
	for id, name := range rock.channels {
		if room == name {
			roomId = id
			break
		}
	}
	if roomId == "" {
		return make([]string, 0), errors.New("No Known Room")
	}
	users, err := rock.ListUsersInRoomId(roomId)
	return users, err
}
