package util

import (
	"strconv"
	"time"
)

type KEY struct {
	ramble int64
}

func (k *KEY) NewKey(length int) string {
	raw := strconv.FormatInt(time.Now().UnixNano()+k.ramble, 16)
	//log.Printf("ramble %d, raw %s\n", k.ramble, raw)
	k.ramble += 1

	switch length {
	case 4, 8, 12:
	default:
		length = 16
	}
	return raw[16-length:]
}

var Key = KEY{0}

type StringStack struct {
	Store map[int]string
	Top   int
}

func NewStringStack() *StringStack {
	return &StringStack{
		Store: make(map[int]string),
		Top:   0,
	}
}

func (p *StringStack) Push(e string) {
	p.Store[p.Top] = e
	p.Top++
}

//No boundary protection, user must ensure.
func (p *StringStack) Pop() string {
	p.Top--
	e, _ := p.Store[p.Top]
	delete(p.Store, p.Top)
	return e
}

func (p *StringStack) IsEmpty() bool {
	if p.Top == 0 {
		return true
	} else {
		return false
	}
}

//No boundary protection, user must ensure.
func (p *StringStack) Get(i int) string {
	e, _ := p.Store[i]
	return e
}
