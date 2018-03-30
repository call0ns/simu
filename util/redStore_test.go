package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var store = NewRedisStore("192.168.209.128:6379")

func TestStoreAndFetchSession(t *testing.T) {
	makeObject()
	s := NewSession("123456789")
	s.SetSubscriptionInfo(2, []string{"invite", "register", "complete"})
	s.Req = &object
	store.StoreSession(s)

	r := store.FetchSession(s.Key)
	assert.Equal(t, s.MsgId, r.MsgId)
	assert.Equal(t, s.Key, r.Key)
	assert.Equal(t, s.Sequence, r.Sequence)
	//	//	assert.Equal(t, s.Handler, r.)
	assert.Equal(t, s.NextHandler, r.NextHandler)
	assert.Equal(t, s.Handler[2], r.Handler[2])

	assert.NotNil(t, s.Req)
	assert.Nil(t, s.Resp)
	assert.Nil(t, s.Err)

	store.RemoveSession(s.Key)
	r = store.FetchSession(s.Key)
	assert.Nil(t, r)

}

func TestStoreAndFetchSessionData(t *testing.T) {
	makeObject()
	s := NewSession("123456789")
	s.SetSubscriptionInfo(2, []string{"invite", "register", "complete"})
	s.Req = &object
	store.StoreSession(s)

	s.NextHandler = 1
	store.StoreSessionData(s, FieldNextHandler)

	r := store.FetchSession(s.Key)
	assert.Equal(t, 1, r.NextHandler)

	s.Err = MakeSipErrorResponse(501, "Not implemented.")
	store.StoreSessionData(s, FieldErr)

	e := store.FetchSessionData(s.Key, FieldErr)
	resp, _ := e.ReadObject("response")
	code, _, _, _ := resp.ReadElementAsString("status")
	assert.Equal(t, "501", code)

}
