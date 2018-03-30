package util

import (
	//	log "repo.oam.ericloud/paas.git/poc2015/util/levlog"
	"net/http"
	//"repo.oam.ericloud/paas.git/poc2015/sipapp/conf"
	"strconv"
	"time"
)

type Storage interface {
	StoreSession(*Session)
	FetchSession(string) *Session
	RemoveSession(string)

	StoreSessionData(*Session, string)           //session, Field
	FetchSessionData(string, string) *JsonObject //key, Field, value
	RemoveSessionData(string, string)            //key, Field
}

const (
	SessionStruct    = "Session"
	FieldKey         = "Key"
	FieldMsgId       = "MsgId"
	FieldSequence    = "Sequence"
	FieldHandler     = "Handler"
	FieldLength      = "Length"
	FieldNextHandler = "NextHandler"
	FieldReq         = "Req"
	FieldResp        = "Resp"
	FieldErr         = "Err"
	FieldTrace       = "Trace"
	FieldTop         = "Top"
	FieldSender      = "Sender"
)

var ramble int64 = 0
var Cache SessionCache = NewSessionCache()
var Store Storage

//Session here is the connection between application orchestrator and handler.
//It keep the information in order to send back response to SIP UA.
type Session struct {
	Key         string // Generated uuid to identify session
	MsgId       string
	Sequence    int      // Message sequence model
	Handler     []string // Names of handlers for processing
	NextHandler int      // Index of next handler
	RespWriter  http.ResponseWriter
	Channel     chan string  // To notify the thread of HTTP Req to release
	Req         *JsonObject  // json object of the SIP Request as process input
	Resp        *JsonObject  // processing result, can be intermediate or final
	Err         *JsonObject  // error information to send to SIPUA
	Trace       *StringStack // store the sender in the stack to return message
	Sender      string
	Born        int64
	// NextHandler, Trace, Resp and Err may be updated during the message processing

}

func NewSession(mid string) *Session {
	key := strconv.FormatInt(time.Now().UnixNano()+ramble, 16)
	ramble++

	return &Session{
		Key:         key,
		MsgId:       mid,
		NextHandler: 0,
		Resp:        nil,
		Err:         nil,
		Trace:       NewStringStack(),
		Born:        time.Now().UnixNano(),
	}
}

func (s *Session) String() string {
	return "Key: " + s.Key + "; Mid: " + s.MsgId
}

func (s *Session) SetRequestInfo(w http.ResponseWriter, r *JsonObject, c chan string) {
	s.RespWriter = w
	s.Req = r
	s.Channel = c
}

func (s *Session) SetSubscriptionInfo(seq int, h []string) {
	s.Sequence = seq
	s.Handler = h
}

func (s *Session) SetSequence(seq int) {
	s.Sequence = seq
}

func (s *Session) StepNextHandler() {
	s.NextHandler++
	Store.StoreSessionData(s, FieldNextHandler)
}

func (s *Session) SetResponse(resp *JsonObject) {
	s.Resp = resp
	Store.StoreSessionData(s, FieldResp)
}

func (s *Session) StoreResponse() {
	Store.StoreSessionData(s, FieldResp)
}

func (s *Session) StoreTrace() {
	Store.StoreSessionData(s, FieldTrace)
}

func (s *Session) SetError(e *JsonObject) {
	s.Err = e
	Store.StoreSessionData(s, FieldErr)
}

func (s *Session) PutToStore() {
	Store.StoreSession(s)
}

func (s *Session) SyncSession() {
	s.Resp = Store.FetchSessionData(s.Key, FieldResp)
	s.Err = Store.FetchSessionData(s.Key, FieldErr)
}

// data structure to keep Key - Session mapping
// can be extended to save to external storage
type SessionCache map[string]*Session

func NewSessionCache() SessionCache {
	return make(map[string]*Session)
}

func (cache *SessionCache) AddSession(s *Session) {
	// to be persist to DB
	(*cache)[(*s).Key] = s
}

func (cache *SessionCache) DelSession(s *Session) {
	// to be persist to DB
	delete(*cache, (*s).Key)
	Store.RemoveSession(s.Key)
}

func (cache *SessionCache) GetSession(key string) *Session {
	s, OK := (*cache)[key]
	if OK == true {
		return s
	} else {
		return Store.FetchSession(key)
	}
}
