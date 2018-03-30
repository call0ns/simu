package util

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	//log "repo.oam.ericloud/paas.git/poc2015/util/levlog"
	"repo.oam.ericloud/paas.git/poc2015/protocol"
	"repo.oam.ericloud/paas.git/poc2015/sipapp_mono/conf"
	"testing"
)

var jsonString = []byte(`{
	"String":"String Value",
	"Integer":12345,
	"Float":1234.567,
	"BoolFalse":false,
	"BoolTrue":true,
	"StringArray3": [ "str0", "str1", "str2" ],
	"StringArray1": [ "str0" ],
	"IntArray4": [ 0, 1, 2, 3 ],
	"IntArray0": [ ],
	"Object": {
		"String":"String Value",
		"Integer":12345,
		"Float":1234.567,
		"BoolFalse":false,
		"BoolTrue":true,
		"StringArray3": [ "str0", "str1", "str2" ],
		"StringArray1": [ "str0" ],
		"IntArray4": [ 0, 1, 2, 3 ],
		"IntArray0": [ ]
	},
	"NOTIFY": "all",
	"MESSAGE": "none"
}`)

type R4 struct { //pack return values in one struct so that can use one assert
	a1 interface{}
	a2 interface{}
	a3 interface{}
	a4 interface{}
}

func makeR4(a1, a2, a3, a4 interface{}) R4 {
	return R4{a1, a2, a3, a4}
}

var object JsonObject

func makeObject() {
	object = NewJsonObject()
	err := json.Unmarshal(jsonString, &object)
	if err != nil {
		panic(err)
	}

}

func TestPrintJasonObject(t *testing.T) {
	//	jobj := NewJsonObject()
	//	err := json.Unmarshal(jsonString, &jobj)
	//	if err != nil {
	//		panic(err)
	//	}

	//	jobj.BrowseJsonObject("=== Test new print ===")

}

func TestAppendPutElement(t *testing.T) {
	object = NewJsonObject()

	object.AppendElement(protocol.HeaderAppId, conf.AppName)
	object.AppendElement(protocol.HeaderAppUri, conf.OwnUrl)

	filter := NewJsonObject()
	filter.AppendElement(protocol.REGISTER, "Contact")
	filter.AppendElement(protocol.REGISTER, "Expires")

	filter.AppendElement(protocol.INVITE, "From")
	filter.AppendElement(protocol.INVITE, "To")
	filter.AppendElement(protocol.INVITE, "Call-ID")
	filter.AppendElement(protocol.NOTIFY, "all")
	filter.PutElement(protocol.MESSAGE, "none")

	object.AppendElement(protocol.HeaderFilter, filter)

	//object.BrowseJsonObject("=== Test new print ===")
	//log.Println(object.ToString())
}

func TestCheckMandatoryElements(t *testing.T) {
	makeObject()
	//log.Println(object.ToString())

	m1 := []string{"NOTIFY", "Object", "MESSAGE"}
	missing, OK := object.CheckMandatoryElements(m1)
	assert.Equal(t, true, OK)
	assert.Nil(t, missing)

	m2 := []string{"NOTIFY", "Object1", "Message"}
	missing, OK = object.CheckMandatoryElements(m2)
	assert.Equal(t, OK, false)
	assert.Equal(t, missing, []string{"Object1", "Message"})

}

func TestReadElementForNestedObject(t *testing.T) {
	makeObject()
	var r4 R4

	var obj JsonObject

	obj, OK := object.ReadObject("Object")
	assert.Equal(t, true, OK)
	assert.NotNil(t, obj)

	r4 = makeR4(obj.ReadElementAsString("String"))
	assert.Equal(t, R4{"String Value", true, -1, TypeString}, r4)

	r4 = makeR4(obj.ReadElementAsString("String"))
	assert.Equal(t, R4{"String Value", true, -1, TypeString}, r4)

	r4 = makeR4(obj.ReadElementAsString("Float"))
	assert.Equal(t, R4{"1234.567", true, -1, TypeFloat64}, r4)

	r4 = makeR4(obj.ReadElementAsString("Integer"))
	assert.Equal(t, R4{"12345", true, -1, TypeInt}, r4)

	r4 = makeR4(obj.ReadElementAsString("BoolFalse"))
	assert.Equal(t, R4{"false", true, -1, TypeBool}, r4)

	r4 = makeR4(obj.ReadElementAsString("BoolTrue"))
	assert.Equal(t, R4{"true", true, -1, TypeBool}, r4)

	r4 = makeR4(obj.ReadElementAsString("StringArray3"))
	assert.Equal(t, R4{"str0", true, 3, TypeString}, r4)

	r4 = makeR4(obj.ReadElementAsString("StringArray1"))
	assert.Equal(t, R4{"str0", true, 1, TypeString}, r4)

	r4 = makeR4(obj.ReadElementAsString("IntArray4"))
	assert.Equal(t, R4{"0", true, 4, TypeInt}, r4)

	r4 = makeR4(obj.ReadElementAsString("IntArray0"))
	assert.Equal(t, R4{"", false, 0, TypeUnknown}, r4)

	obj.AppendElement("IntArray4", 9223372036854775807)
	obj.AppendElement("IntArray4", 5)

	r4 = makeR4(obj.ReadElementAsString("IntArray4"))
	assert.Equal(t, R4{"0", true, 6, TypeInt}, r4)

	r4 = makeR4(obj.ReadElementAsStringFromList("IntArray4", 4))
	assert.Equal(t, R4{"9223372036854775807", true, 6, TypeInt}, r4)

	//object.BrowseJsonObject("=== check update result ===")
	//log.Println(obj.ToString())
}

func TestReadElementArrayByIndex(t *testing.T) {
	makeObject()
	var r4 R4

	r4 = makeR4(object.ReadElementAsStringFromList("StringArray3", 2))
	assert.Equal(t, R4{"str2", true, 3, TypeString}, r4)

	r4 = makeR4(object.ReadElementAsStringFromList("IntArray4", 4))
	assert.Equal(t, R4{"", false, 4, TypeUnknown}, r4)

}

func TestReadElementArray(t *testing.T) {
	makeObject()
	var r4 R4

	r4 = makeR4(object.ReadElementAsString("StringArray3"))
	assert.Equal(t, R4{"str0", true, 3, TypeString}, r4)

	r4 = makeR4(object.ReadElementAsString("StringArray1"))
	assert.Equal(t, R4{"str0", true, 1, TypeString}, r4)

	r4 = makeR4(object.ReadElementAsString("IntArray4"))
	assert.Equal(t, R4{"0", true, 4, TypeInt}, r4)

	r4 = makeR4(object.ReadElementAsString("IntArray0"))
	assert.Equal(t, R4{"", false, 0, TypeUnknown}, r4)

}

func TestReadElementSingle(t *testing.T) {
	makeObject()
	var r4 R4

	r4 = makeR4(object.ReadElementAsString("String"))
	assert.Equal(t, R4{"String Value", true, -1, TypeString}, r4)

	r4 = makeR4(object.ReadElementAsString("Float"))
	assert.Equal(t, R4{"1234.567", true, -1, TypeFloat64}, r4)

	r4 = makeR4(object.ReadElementAsString("Integer"))
	assert.Equal(t, R4{"12345", true, -1, TypeInt}, r4)

	r4 = makeR4(object.ReadElementAsString("BoolFalse"))
	assert.Equal(t, R4{"false", true, -1, TypeBool}, r4)

	r4 = makeR4(object.ReadElementAsString("BoolTrue"))
	assert.Equal(t, R4{"true", true, -1, TypeBool}, r4)

	r4 = makeR4(object.ReadElementAsString("Object"))
	assert.Equal(t, R4{"", false, -1, TypeObject}, r4)

}

func TestReadElement(t *testing.T) {
	makeObject()
	var r4 R4

	r4 = makeR4(object.ReadElement("String"))
	assert.Equal(t, R4{"String Value", true, -1, TypeString}, r4)

	r4 = makeR4(object.ReadElement("Float"))
	assert.Equal(t, R4{1234.567, true, -1, TypeFloat64}, r4)

	a1, _, _, _ := object.ReadElement("Integer")
	assert.Equal(t, int64(12345), a1)

	r4 = makeR4(object.ReadElement("Integer"))
	assert.Equal(t, R4{int64(12345), true, -1, TypeInt}, r4)

	r4 = makeR4(object.ReadElement("BoolFalse"))
	assert.Equal(t, R4{false, true, -1, TypeBool}, r4)

	r4 = makeR4(object.ReadElement("BoolTrue"))
	assert.Equal(t, R4{true, true, -1, TypeBool}, r4)

	r4 = makeR4(object.ReadElement("Object"))
	o, _ := object.ReadObject("Object")
	assert.Equal(t, R4{o, true, -1, TypeObject}, r4)

	r4 = makeR4(object.ReadElement("IntArray4"))
	assert.Equal(t, R4{int64(0), true, 4, TypeInt}, r4)

	r4 = makeR4(object.ReadElementFromList("IntArray4", 2))
	assert.Equal(t, R4{int64(2), true, 4, TypeInt}, r4)

}

func TestAddEmptyArray(t *testing.T) {
	obj := NewJsonObject()
	//	var strArray []string
	//	strArray = append(strArray, "str1")
	//	strArray = append(strArray, "str2")
	obj.AddEmptyArray("StringArray")
	_, OK, length, _ := obj.ReadElementAsString("StringArray")
	assert.Equal(t, false, OK)
	assert.Equal(t, 0, length)
	obj.AppendElement("StringArray", "Str1")
	_, OK, length, _ = obj.ReadElementAsString("StringArray")
	assert.Equal(t, true, OK)
	assert.Equal(t, 1, length)

}
