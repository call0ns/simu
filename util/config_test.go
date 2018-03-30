//Some tests may fail because environment variable not set in OS env.
package util

import (
	"github.com/stretchr/testify/assert"
	log "repo.oam.ericloud/paas.git/poc2015/util/levlog"
	"strconv"
	"testing"
)

var jsonString1 = []byte(`{
	"Checksum": "123456",
	"Parameters": {
		"String":"nats://10.0.0.1:4222",
		"Integer":"12345",
		"Float":"1234.567",
		"Bool":"false"
	}
}`)

var jsonString2 = []byte(`{
	"Checksum": "123456",
	"Parameters": {
		"String":"nats://10.0.0.1:8222",
		"Integer":"108",
		"Float":"1234.567",
		"Bool":"true"
	}
}`)

var jsonString3 = []byte(`{
	"Checksum": "123456",
	"Parameters": {
		"String":"nats://10.0.0.1:8222",
		"Integer":"108",
		"Float":"1234.567",
		"Bool":"false",
		"Any Parameter":"Anything"
	}
}`)

var jsCollect = []byte(`{
	"Parameters":["String","Integer","Bool","Float","Others"]
}`)

func init() {
	log.Start(3)

}

func TestCMUseCases(t *testing.T) {
	obj1, err1 := Bytes2Json(jsonString1)
	if err1 != nil {
		panic(err1)
	}
	obj2, err2 := Bytes2Json(jsonString2)
	if err2 != nil {
		panic(err2)
	}
	obj3, err3 := Bytes2Json(jsonString3)
	if err2 != nil {
		panic(err3)
	}
	obj4, err4 := Bytes2Json(jsCollect)
	if err2 != nil {
		panic(err4)
	}
	conf := NewConfiguration()
	pString := conf.NewOSEnvParameter("String", "nats", "nats://192.168.209.128:8222")
	assert.Equal(t, TypeString, pString.Type())
	pString.SetApplyFunction(applyNewParameter)

	pInt := conf.NewParameter("Integer", 1000)
	assert.Equal(t, TypeInt64, pInt.Type())
	pInt.SetValidateFunction(validateIntParam)
	pInt.SetApplyFunction(applyNewParameter)

	pFloat := conf.NewParameter("Float", 3456789.9876543)
	assert.Equal(t, TypeFloat64, pFloat.Type())
	pFloat.SetApplyFunction(applyNewParameter)

	pBool := conf.NewParameter("Bool", true)
	assert.Equal(t, TypeBool, pBool.Type())
	pBool.SetApplyFunction(applyNewParameter)

	OK := pInt.Validate("12345")
	assert.Equal(t, "Value (12345) must not greater than 1000", OK)

	OK = pInt.Validate("108")
	assert.Equal(t, "", OK)

	OK = pInt.Validate("abcn,yt")
	assert.Equal(t, "Value (abcn,yt) cannot be parsed into integer.", OK)

	//validation not approved, reject due to checksum not match
	conf.setChecksum("1234")
	res, fresh := conf.Validate(obj1)
	assert.Equal(t, false, fresh)

	//validation not approved, reject by value checking
	conf.setChecksum("123456")
	res, fresh = conf.Validate(obj1)
	approve, _, _, _ := res.ReadElementAsString("Result")
	assert.Equal(t, true, fresh)
	assert.Equal(t, "Disapprove", approve)
	assert.Equal(t, "123456", conf.checksum)

	//validation approved
	res, _ = conf.Validate(obj2)
	approve, _, _, _ = res.ReadElementAsString("Result")
	assert.Equal(t, "Approve", approve)

	//validation approved
	res, _ = conf.Validate(obj3)
	approve, _, _, _ = res.ReadElementAsString("Result")
	assert.Equal(t, "Approve", approve)
	unknown, _, _, _ := res.ReadElementAsString("Unknown")
	assert.Equal(t, "Any Parameter", unknown)

	//initial apply
	conf.setChecksum("")
	conf.Apply(obj2)
	assert.Equal(t, "123456", conf.checksum)
	assert.Equal(t, "nats://10.0.0.1:8222", pString.Get().(string))

	//validation pass
	//OK = conf.Validate(obj3)
	OK = pInt.Validate("108")
	assert.Equal(t, "", OK)

	//checksum no change, skip apply
	conf.setChecksum("123456")
	conf.Apply(obj3)
	assert.Equal(t, "123456", conf.checksum)
	assert.Equal(t, true, pBool.Get().(bool))

	//checksum change, new value applied
	conf.setChecksum("3456")
	conf.Apply(obj3)
	assert.Equal(t, "123456", conf.checksum)
	assert.Equal(t, false, pBool.Get().(bool))

	conf.Collect(obj4).BrowseJsonObject("-- Collect --")

}

func validateIntParam(v string, p *PARAMETER) string {
	val, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return "Value (" + v + ") cannot be parsed into integer."
	}
	if val > 1000 {
		return "Value (" + v + ") must not greater than 1000"
	}
	if val < 100 {
		return "Value (" + v + ") must not less than 100"
	}
	return ""
}

func applyNewParameter(p *PARAMETER) {
	log.Printf("Par[%s].(%s):", p.Name, p.Type())
	log.Println("Prev=", p.prev, " new=", p.value)
}

func TestConfig(t *testing.T) {
	conf := NewConfiguration()

	p1 := conf.NewOSEnvParameter("nats", "nats", "nats://192.168.209.128:8222")
	assert.Equal(t, "string", p1.Type())

	var nats string
	nats = p1.Get().(string)
	assert.Equal(t, "nats://192.168.209.128:8222", nats)
	assert.Equal(t, "nats://192.168.209.128:8222", p1.Get().(string))

	s, _ := conf.SafeGet("nats")
	assert.Equal(t, "nats://192.168.209.128:8222", s)

	p2 := conf.NewOSEnvParameter("app_id", "app_id", "MyApp")
	assert.Equal(t, "SIPAPP", p2.Get())
	assert.Equal(t, "SIPAPP", conf.Get("app_id"))

	p3 := conf.NewOSEnvParameter("proc", "PROCESSOR_LEVEL", 8)
	assert.Equal(t, "int64", p3.Type())
	assert.Equal(t, int64(6), p3.Get().(int64))
	assert.Equal(t, int64(6), conf.Get("proc"))

	p4 := conf.NewOSEnvParameter("float", "float", 8.01)
	assert.Equal(t, "float64", p4.Type())

	assert.Equal(t, float64(8.01), p4.Get())

	f, _ := conf.SafeGet("float")
	assert.Equal(t, float64(8.01), f)

	p5 := conf.NewOSEnvParameter("bool", "bool", true)
	assert.Equal(t, "bool", p5.Type())

	assert.Equal(t, true, p5.Get())
	b, _ := conf.SafeGet("bool")
	assert.Equal(t, true, b)

	_, OK := conf.SafeGet("bool1")
	assert.Equal(t, false, OK)

	p6 := conf.NewParameter("nats", "abcdefg")
	assert.Nil(t, p6)

	p7 := conf.GetParameter("nats")
	assert.Equal(t, "nats://192.168.209.128:8222", p7.Get())

	p8 := conf.NewParameter("param", "abcd")
	assert.Equal(t, "abcd", p8.Get())
}
