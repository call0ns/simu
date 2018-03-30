package util

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	//log "repo.oam.ericloud/paas.git/poc2015/util/levlog"
	//"repo.oam.ericloud/paas.git/poc2015/sipapp_mono/conf"
	"strconv"
)

//	type Storage interface
//		StoreSession(*Session)
//		FetchSession(string)*Session //key
//		RemoveSession(string)//Key
//		StoreSessionData(*Session, string)//session, Field
//		FetchSessionData(string, string)string //key, Field, value
//		RemoveSessionData(string, string)//key, Field

type RedisStore struct {
	pool *redis.Pool
}

var app_id string

func (r *RedisStore) GetPool() *redis.Pool {
	return r.pool
}

func SetAppName(id string) {
	//log.Printf("Application ID is %s\n", id)
	app_id = id
}

func NewRedisStore(uri string) *RedisStore {
	return &RedisStore{
		pool: newPool(uri),
	}
}

func NewRedisStoreWithPool(p *redis.Pool) *RedisStore {
	return &RedisStore{
		pool: p,
	}
}

func newPool(uri string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:   80,
		MaxActive: 100, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", uri)
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}

}

func (r *RedisStore) StoreSession(s *Session) {
	c := r.pool.Get()
	defer c.Close()
	redObject := app_id + "." + SessionStruct + "." + s.Key

	c.Do("HSET", redObject, FieldKey, s.Key)
	c.Do("HSET", redObject, FieldMsgId, s.MsgId)
	c.Do("HSET", redObject, FieldSender, s.Sender)
	c.Do("HSET", redObject, FieldSequence, s.Sequence)
	c.Do("HSET", redObject, FieldNextHandler, s.NextHandler)

	c.Do("HSET", redObject, FieldHandler+"."+FieldLength, len(s.Handler))
	for i, v := range s.Handler {
		c.Do("HSET", redObject, FieldHandler+"."+strconv.Itoa(i), v)
	}

	//	c.Do("HSET", redObject, FieldTrace+"."+FieldTop, s.Trace.Top)
	//	for i := 0; i < s.Trace.Top; i++ {
	//		c.Do("HSET", redObject, FieldTrace+"."+strconv.Itoa(i), s.Trace.Get(i))
	//	}

	if s.Req != nil {
		req, _ := json.Marshal(s.Req)
		c.Do("HSET", redObject, FieldReq, req)
	}
	if s.Resp != nil {
		resp, _ := json.Marshal(s.Resp)
		c.Do("HSET", redObject, FieldResp, resp)
	}
	if s.Err != nil {
		err, _ := json.Marshal(s.Err)
		c.Do("HSET", redObject, FieldErr, err)
	}
	c.Do("EXPIRE", redObject, 60)
}

func (r *RedisStore) FetchSession(key string) *Session {
	//log.Printf("Fetch session [%s] from storage.", key)
	c := r.pool.Get()
	defer c.Close()
	redObject := app_id + "." + SessionStruct + "." + key
	OK, _ := redis.Int(c.Do("EXISTS", redObject))
	if OK == 0 {
		return nil
	}

	key1, _ := redis.String(c.Do("HGET", redObject, FieldKey))
	mid, _ := redis.String(c.Do("HGET", redObject, FieldMsgId))
	sender, _ := redis.String(c.Do("HGET", redObject, FieldSender))
	seq, _ := redis.Int(c.Do("HGET", redObject, FieldSequence))
	nxt, _ := redis.Int(c.Do("HGET", redObject, FieldNextHandler))
	hlen, _ := redis.Int(c.Do("HGET", redObject, FieldHandler+"."+FieldLength))

	var handlers []string
	for i := 0; i < hlen; i++ {
		h, _ := redis.String(c.Do("HGET", redObject, FieldHandler+"."+strconv.Itoa(i)))
		handlers = append(handlers, h)
	}

	//	top, _ := redis.Int(c.Do("HGET", redObject, FieldTrace+"."+FieldTop))
	//	stack := NewStringStack()
	//	for i := 0; i < top; i++ {
	//		h, _ := redis.String(c.Do("HGET", redObject, FieldTrace+"."+strconv.Itoa(i)))
	//		stack.Push(h)
	//	}

	reqp := hgetJsonObj(c, redObject, FieldReq)
	//	respp := hgetJsonObj(c, redObject, FieldResp)
	//	errp := hgetJsonObj(c, redObject, FieldErr)

	return &Session{
		Key:         key1,
		MsgId:       mid,
		Sequence:    seq,
		NextHandler: nxt,
		Handler:     handlers,
		Req:         reqp,
		//		Resp:        respp,
		//		Err:         errp,
		//		Trace:       stack,
		Sender: sender,
	}
}

func (r *RedisStore) RemoveSession(key string) {
	c := r.pool.Get()
	defer c.Close()
	redObject := app_id + "." + SessionStruct + "." + key

	c.Do("DEL", redObject)
}

func (r *RedisStore) StoreSessionData(s *Session, f string) {
	c := r.pool.Get()
	defer c.Close()
	redObject := app_id + "." + SessionStruct + "." + s.Key
	OK, _ := redis.Int(c.Do("EXISTS", redObject))
	if OK == 0 {
		return
	}

	switch f {
	case FieldNextHandler:
		c.Do("HSET", redObject, FieldNextHandler, s.NextHandler)
	case FieldReq:
		req, _ := json.Marshal(s.Req)
		c.Do("HSET", redObject, FieldReq, req)
	case FieldResp:
		resp, _ := json.Marshal(s.Resp)
		c.Do("HSET", redObject, FieldResp, resp)
	case FieldErr:
		err, _ := json.Marshal(s.Err)
		c.Do("HSET", redObject, FieldErr, err)
	case FieldHandler:
		c.Do("HSET", redObject, FieldHandler+"."+FieldLength, len(s.Handler))
		for i, v := range s.Handler {
			c.Do("HSET", redObject, FieldHandler+"."+strconv.Itoa(i), v)
		}
	case FieldTrace:
		c.Do("HSET", redObject, FieldTrace+"."+FieldTop, s.Trace.Top)
		for i := 0; i < s.Trace.Top; i++ {
			c.Do("HSET", redObject, FieldTrace+"."+strconv.Itoa(i), s.Trace.Get(i))
		}
	default:
	}
}

func (r *RedisStore) FetchSessionData(key string, f string) (v *JsonObject) {
	c := r.pool.Get()
	defer c.Close()
	redObject := app_id + "." + SessionStruct + "." + key
	OK, _ := redis.Int(c.Do("EXISTS", redObject))
	if OK == 0 {
		return nil
	}

	return hgetJsonObj(c, redObject, f)

}

func hgetJsonObj(c redis.Conn, key string, Field string) *JsonObject {
	str, _ := redis.String(c.Do("HGET", key, Field))
	if str != "" {
		obj := NewJsonObject()
		json.Unmarshal([]byte(str), &obj)
		return &obj
	} else {
		return nil
	}
}

func HGetJsonObject(c redis.Conn, key string, field string) *JsonObject {
	return hgetJsonObj(c, key, field)
}

func (r *RedisStore) RemoveSessionData(key string, f string) {
	c := r.pool.Get()
	defer c.Close()
	redObject := app_id + "." + SessionStruct + "." + key
	OK, _ := redis.Int(c.Do("EXISTS", redObject))
	if OK == 0 {
		return
	}

	c.Do("HDEL", redObject, f)

}
