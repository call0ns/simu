package util

import (
	"github.com/stretchr/testify/assert"
	//log "repo.oam.ericloud/paas.git/poc2015/util/levlog"
	"testing"
)

func TestSession(t *testing.T) {
	s := NewSession("1234")
	assert.Equal(t, "1234", s.MsgId)
	//	log.Println(s.String())

}

func TestSessionStore(t *testing.T) {
	cache := NewSessionCache()
	Store = NewRedisStore("192.168.209.128:6379")

	s1 := NewSession("1234")
	s1.PutToStore()
	s2 := NewSession("1235")
	s2.PutToStore()
	s3 := NewSession("1236")
	s3.PutToStore()
	cache.AddSession(s1)
	cache.AddSession(s2)
	cache.AddSession(s3)

	s := NewSession("1237")
	s.SetSubscriptionInfo(2, []string{"invite", "register", "complete"})
	s.PutToStore()
	cache.AddSession(s)

	s4 := NewSession("1236")
	s4.PutToStore()
	cache.AddSession(s4)

	r1 := cache.GetSession(s.Key)
	assert.Equal(t, "1237", r1.MsgId)

	cache.DelSession(r1)
	r2 := cache.GetSession(r1.Key)
	assert.Nil(t, r2)

}
