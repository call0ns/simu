package util

import (
	"encoding/json"
	"repo.oam.ericloud/paas.git/poc2015/protocol"
	log "repo.oam.ericloud/paas.git/poc2015/util/levlog"
	"strconv"
	"strings"
)

const (
	ROOT = "root"
)

const (
	TypeInt     = "int"
	TypeInt64   = "int64"
	TypeFloat64 = "float64"
	TypeString  = "string"
	TypeBool    = "bool"
	TypeObject  = "object"
	TypeUnknown = "unknown"
	TypeNil     = "nil"
)

type JsonObject map[string]interface{}

func NewJsonObject() JsonObject {
	return make(map[string]interface{})
}

func NewJsonObjectPointer() *JsonObject {
	a := JsonObject(make(map[string]interface{}))
	return &a
}

func (obj *JsonObject) BrowseJsonObject(title string) {
	log.Debugcd(1, "___ Start of Object ____________")
	obj.printJsonObject(title, 0)
	log.Debugcd(1, "___ End of Object ______________")
}

func Json2Bytes(js JsonObject) []byte {
	data, _ := json.Marshal(js)
	return data
}

func Bytes2Json(data []byte) (*JsonObject, error) {
	js := NewJsonObject()
	err := json.Unmarshal(data, &js)
	return &js, err
}

func MakeSipErrorResponse(code int, text string) *JsonObject {
	err := NewJsonObject()
	err.AppendElement(protocol.HeaderStatus, code)
	err.AppendElement(protocol.HeaderStatusText, text)

	resp := NewJsonObject()
	resp.AppendElement(protocol.RESPONSE, err)

	log.Infofcd(1, "%d %s", code, text)
	return &resp
}

func (obj *JsonObject) printJsonObject(title string, cnt int) {
	var tab string = ""
	for i := 0; i < cnt; i++ {
		tab = tab + "  "
	}
	log.Debugcd(1, tab, title)

	for k, v := range *obj {
		switch vv := v.(type) {
		case string:
			log.Debugcd(1, tab, k, " : ", vv)
		case bool:
			log.Debugcd(1, tab, k, " : ", vv)
		case float64:
			log.Debugcd(1, tab, k, " : ", vv)
		case int:
			log.Debugcd(1, tab, k, " : ", vv)
		case []interface{}:
			log.Debugcd(1, tab, k, " : ")
			for i, u := range vv {
				log.Debugcd(1, tab, "    ", i, u)
			}
		case map[string]interface{}:
			object := v.(map[string]interface{})
			var o JsonObject = JsonObject(object)
			//cnt = cnt + 1
			o.printJsonObject("=== Object : "+k+" ===", cnt+1)

		case JsonObject:
			var o JsonObject = v.(JsonObject)
			//cnt = cnt + 1
			o.printJsonObject("=== Object : "+k+" ===", cnt+1)
		default:
			log.Debugcd(1, k, "is unknown type.")
		}
	}
}

func (obj *JsonObject) CheckMandatoryElements(names []string) ([]string, bool) {
	var missing []string
	OK := true

	for _, name := range names {
		_, r := (*obj)[name]
		if r == false {
			missing = append(missing, name)
			OK = false
		}
	}

	return missing, OK
}

func (obj *JsonObject) DeleteElement(name string) {
	delete(*obj, name)

}

func DeleteElementInArray(object *map[string]interface{}, name string, idx int) {

}

func (obj *JsonObject) ReadElementAsStringFromList(name string, idx int) (string, bool, int, string) {
	//element must be array otherwise return false
	var v string = ""
	var t string = TypeUnknown
	more := 0

	any, OK := (*obj)[name]
	if OK == false {
		return v, false, more, t
	}

	var e interface{}

	switch anyType := any.(type) {
	case []interface{}:
		more = len(anyType)
		if more > 0 {
			if idx >= more {
				OK = false
			} else {
				e = anyType[idx]
			}
		} else {
			OK = false
		}

	default:
		OK = false
	}

	if OK == false {
		return v, OK, more, t
	}

	switch eType := e.(type) {
	case string:
		v = eType
		t = TypeString

	case bool:
		b := eType
		v = strconv.FormatBool(b)
		t = TypeBool

	case float64:
		var f float64 = eType
		v = strconv.FormatFloat(f, 'f', -1, 64)
		if strings.Contains(v, ".") {
			t = TypeFloat64
		} else {
			t = TypeInt
		}
	case int:
		var i int = eType
		v = strconv.Itoa(i)
		t = TypeInt

	default:
		log.Println("Error 6 - ", eType)
		OK = false
	}
	return v, OK, more, t

}

func (obj *JsonObject) ReadElementAsString(name string) (string, bool, int, string) {
	var v string = ""
	var t string = TypeUnknown
	// if the element is array, more is the length of the array
	more := -1

	any, OK := (*obj)[name]
	if OK == false {
		return v, false, more, t
	}

	var e interface{}

	switch anyType := any.(type) {
	case string:
		v = anyType
		t = TypeString

	case bool:
		b := anyType
		v = strconv.FormatBool(b)
		t = TypeBool

	case int:
		var i int = anyType
		v = strconv.Itoa(i)
		t = TypeInt

	case float64:
		var f float64 = anyType
		v = strconv.FormatFloat(f, 'f', -1, 64)
		if strings.Contains(v, ".") {
			t = TypeFloat64
		} else {
			t = TypeInt
		}

	case []interface{}:
		more = len(anyType)
		if more > 0 {
			e = anyType[0]
		} else {
			OK = false
		}

	case map[string]interface{}:
		OK = false
		t = TypeObject

	default:
		OK = false
	}

	if more <= 0 {
		return v, OK, more, t
	}

	switch eType := e.(type) {
	case string:
		v = eType
		t = TypeString

	case bool:
		b := eType
		v = strconv.FormatBool(b)
		t = TypeBool

	case float64:
		var f float64 = eType
		v = strconv.FormatFloat(f, 'f', -1, 64)
		if strings.Contains(v, ".") {
			t = TypeFloat64
		} else {
			t = TypeInt
		}

	default:
		OK = false
	}
	return v, OK, more, t

}

func (obj *JsonObject) ReadElement(name string) (interface{}, bool, int, string) {

	var t string = TypeUnknown
	// if the element is array, more is the length of the array
	more := -1

	any, OK := (*obj)[name]
	if OK == false {
		return any, false, more, t
	}

	var e interface{}

	switch anyType := any.(type) {
	case string:
		t = TypeString

	case bool:
		t = TypeBool

	case int:
		t = TypeInt

	case int64:
		t = TypeInt64

	case float64:
		var f float64 = anyType
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if strings.Contains(s, ".") {
			t = TypeFloat64
			any = f
		} else {
			t = TypeInt
			any, _ = strconv.ParseInt(s, 10, 64)
		}

	case []interface{}:
		more = len(anyType)
		if more > 0 {
			e = anyType[0]
		} else {
			OK = false
		}

	case JsonObject:
		t = TypeObject

	case map[string]interface{}:
		t = TypeObject
		//		o := any.(map[string]interface{})
		any = JsonObject(anyType)

	default:
		OK = false
		log.Println("Unknown type: ", anyType)
	}

	if more <= 0 {
		return any, OK, more, t
	}

	switch eType := e.(type) {
	case string:
		t = TypeString

	case int:
		t = TypeInt

	case int64:
		t = TypeInt64

	case bool:
		t = TypeBool

	case float64:
		var f float64 = eType
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if strings.Contains(s, ".") {
			t = TypeFloat64
			e = f
		} else {
			t = TypeInt
			e, _ = strconv.ParseInt(s, 10, 64)
		}

	default:
		OK = false
	}
	return e, OK, more, t

}

func (obj *JsonObject) ReadElementFromList(name string, idx int) (interface{}, bool, int, string) {
	//element must be array otherwise return false
	var v interface{}
	var t string = TypeUnknown
	more := 0

	any, OK := (*obj)[name]
	if OK == false {
		return v, false, more, t
	}

	var e interface{}

	switch anyType := any.(type) {
	case []interface{}:
		more = len(anyType)
		if more > 0 {
			if idx >= more {
				OK = false
			} else {
				e = anyType[idx]
			}
		} else {
			OK = false
		}

	default:
		OK = false
	}

	if OK == false {
		return v, OK, more, t
	}

	switch eType := e.(type) {
	case string:
		t = TypeString

	case bool:
		t = TypeBool

	case float64:
		var f float64 = eType
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if strings.Contains(s, ".") {
			t = TypeFloat64
			e = f
		} else {
			t = TypeInt
			e, _ = strconv.ParseInt(s, 10, 64)
		}
	case int:
		t = TypeInt

	case int64:
		t = TypeInt64

	default:
		log.Println("Error 6 - ", eType)
		OK = false
	}
	return e, OK, more, t

}

func (obj *JsonObject) ReadObject(name string) (JsonObject, bool) {
	any, OK := (*obj)[name]
	if OK == false {
		return nil, false
	}

	switch anyType := any.(type) {
	case map[string]interface{}:
		o := any.(map[string]interface{})
		return JsonObject(o), true
	case JsonObject:
		o := any.(JsonObject)
		return o, true
	default:
		log.Println(anyType, " is not an object!")
		return nil, false
	}

}

func (obj *JsonObject) PutElement(name string, value interface{}) {
	// Compare to AddHeader, if element "name" exsts, it will be overwritten by the new value
	(*obj)[name] = value
}

func (obj *JsonObject) AddEmptyArray(name string) {
	a := make([]interface{}, 0)
	(*obj)[name] = a
}

func (obj *JsonObject) AppendElement(name string, value interface{}) error {
	// Add "name":"value"
	// If element "name" already exists
	//    1 element is array,
	//        1.1 value is single, append value to the array
	//        1.2 value is array, merge the two arrays
	//    2 element is single
	//        2.1 value is single, make new array and put both values into the array
	//        2.2 value is array, append existing value to the array
	v, OK := (*obj)[name]
	if OK == false {
		//new element, directly put
		(*obj)[name] = value
		return nil
	}

	// element exists, check type of it
	switch vv := v.(type) {
	case []interface{}:
		//it is an array
		switch ww := value.(type) {
		case []interface{}: // merge array
			for _, x := range ww {
				vv = append(vv, x)
			}
			(*obj)[name] = vv

		default:
			vv = append(vv, value)
			(*obj)[name] = vv

		}
	default:
		//it is not array
		switch ww := value.(type) {
		case []interface{}: // merge array
			value = append(ww, v)
			(*obj)[name] = ww

		default:
			var newArray []interface{}
			newArray = append(newArray, vv)
			newArray = append(newArray, ww)
			(*obj)[name] = newArray

		}

	}
	return nil
}

func (obj *JsonObject) ToString() string {
	var result = "{"
	addComa := false

	for k, v := range *obj {
		if addComa {
			result = result + ", "
		} else {
			addComa = true
		}
		result = result + `"` + k + `" : `

		switch vv := v.(type) {
		case string:
			result = result + `"` + v.(string) + `"`
		case bool:
			result = result + strconv.FormatBool(vv)
		case float64:
			result = result + strconv.FormatFloat(vv, 'f', -1, 64)
		case int:
			result = result + strconv.Itoa(vv)
		case []interface{}:
			result = result + `[`
			coma := false
			for _, u := range vv {
				if coma {
					result = result + `, `
				} else {
					coma = true
				}
				switch uu := u.(type) {
				case string:
					result = result + `"` + u.(string) + `"`
				case bool:
					result = result + strconv.FormatBool(uu)
				case int:
					result = result + strconv.Itoa(uu)
				case float64:
					result = result + strconv.FormatFloat(uu, 'f', -1, 64)
				default:
				}
			}
			result = result + "]"
		case map[string]interface{}:
			object := v.(map[string]interface{})
			var o JsonObject = JsonObject(object)

			r := o.ToString()
			//			log.Println("R1 --", result)
			//			log.Println("r1 --", r)
			result = result + r

		case JsonObject:
			var o JsonObject = v.(JsonObject)
			result = result + o.ToString()

			o.ToString()
			//			log.Println("R2 --", result)
			//			log.Println("r2 --", r)

		default:
			log.Println(k, "is unknown type.")
		}
	}
	result = result + "}"
	return result
}

func BrowseJasonStructure(f map[string]interface{}, title string) {
	log.Printf("%s\n", title)

	for k, v := range f {
		switch vv := v.(type) {
		case string:
			log.Println(k, " : ", vv)
		case bool:
			log.Println(k, " : ", vv)
		case float64:
			log.Println(k, " : ", vv)
		case []interface{}:
			log.Println(k, " : ")
			for i, u := range vv {
				log.Println("    ", i, u)
			}
		case map[string]interface{}:
			object := v.(map[string]interface{})
			BrowseJasonStructure(object, "=== Object : "+k+" ===")

		default:
			log.Println(k, "is unknown type.")
		}
	}
}
