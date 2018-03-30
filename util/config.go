package util

import (
	"os"
	log "repo.oam.ericloud/paas.git/poc2015/util/levlog"
	"strconv"
)

const (
	EQUAL   = 0
	UNEQUAL = 1
	ERROR   = 2
)

type CONFIGURATION struct {
	params   map[string]*PARAMETER
	checksum string
}

func NewConfiguration() *CONFIGURATION {
	return &CONFIGURATION{
		params:   make(map[string]*PARAMETER),
		checksum: "",
	}
}

func (p *CONFIGURATION) IsFresh(check string) bool {
	if p.checksum == check {
		return true
	} else {
		return false
	}
}

func (p *CONFIGURATION) setChecksum(check string) {
	p.checksum = check
}

//Parameter from CM service is always quoted "string"
//Need to transfer to the right type
func (p *CONFIGURATION) Apply(js *JsonObject) {
	var check string
	var OK bool
	var params JsonObject
	if check, OK, _, _ = js.ReadElementAsString("Checksum"); OK != true {
		log.Error("Bad message format. [Checksum] missing.")
		return
	}
	if p.checksum == check {
		log.Debug("Checksum not change, no need to apply new values.")
		return
	}
	if params, OK = js.ReadObject("Parameters"); OK != true {
		log.Error("Bad message format. [Parameters] missing.")
		return
	}
	for name, value := range params {
		log.Debug("Checking ", name, ":", value)
		param := p.GetParameter(name)
		if param != nil {
			if param.Equal(value.(string)) == UNEQUAL {
				param.Set(value.(string))
				param.apply(param)
			}
		}
	}
	p.setChecksum(check)
}

func (p *CONFIGURATION) Validate(js *JsonObject) (*JsonObject, bool) {
	resp := NewJsonObject()
	reason := NewJsonObject()

	var fresh, OK bool
	var check string
	var params JsonObject

	if check, OK, _, _ = js.ReadElementAsString("Checksum"); OK != true {
		log.Error("Bad message format. [Checksum] missing.")
		return nil, false
	}
	if p.IsFresh(check) {
		if params, OK = js.ReadObject("Parameters"); OK != true {
			log.Error("Bad message format. [Parameters] missing.")
			return nil, false
		}
		fresh = true

		fail := 0
		unknown := 0
		for name, value := range params {
			param := p.GetParameter(name)
			if param == nil {
				if unknown == 0 {
					resp.AddEmptyArray("Unknown")
					unknown++
				}
				resp.AppendElement("Unknown", name)
			} else {
				result := param.Validate(value.(string))
				if len(result) > 0 {
					reason.AppendElement(name, result)
					fail++
				}
			}
		}
		if fail == 0 {
			resp.AppendElement("Result", "Approve")
		} else {
			resp.AppendElement("Result", "Disapprove")
			resp.AppendElement("Reason", reason)
		}
		return &resp, fresh

	} else {
		//This instance doesn't have fresh data, disapprove.
		fresh = false
		resp := NewJsonObject()
		reason := NewJsonObject()
		reason.AppendElement("Others", "This instance doesn't have the fresh configuration data. Please retry later.")
		resp.AppendElement("Result", "Disapprove")
		resp.AppendElement("Reason", reason)

		return &resp, fresh
	}
}

func (p *CONFIGURATION) Collect(js *JsonObject) *JsonObject {
	var OK bool
	var length int
	if _, OK, length, _ = js.ReadElement("Parameters"); OK != true {
		log.Error("Bad message format. [Parameters] missing or emptry.")
		return nil
	}
	if length == 0 {
		log.Error("Bad message format. Empty [Parameter] array.")
		return nil
	}
	params := NewJsonObject()
	resp := NewJsonObject()

	unknown := 0
	for i := 0; i < length; i++ {
		name, _, _, _ := js.ReadElementAsStringFromList("Parameters", i)
		par := p.GetParameter(name)
		if par != nil {
			params.AppendElement(name, par.GetAsString())

		} else { // Parameter not exists.
			log.Errorf("Parameter [%s] doesn't exist in my configuration.", name)
			if unknown == 0 {
				resp.AddEmptyArray("Unknown")
			}
			unknown++
			resp.AppendElement("Unknown", name)
		}
	}
	resp.AppendElement("Parameters", params)
	return &resp
}

func (p *CONFIGURATION) Exists(name string) bool {
	_, exists := p.params[name]
	return exists
}

func (p *CONFIGURATION) AddParameter(par *PARAMETER) {
	p.params[par.Name] = par

}

func (p *CONFIGURATION) GetParameter(name string) *PARAMETER {
	if p.Exists(name) {
		return p.params[name]
	} else {
		return nil
	}
}

func (p *CONFIGURATION) SafeGet(name string) (interface{}, bool) {
	par, OK := p.params[name]
	if OK {
		return par.value, OK
	} else {
		return nil, OK
	}
}

//This function doesn't return validity info. User must ensure that
//  the parameter do exist!
func (p *CONFIGURATION) Get(name string) interface{} {
	par, _ := p.params[name]
	return par.value
}

type PARAMETER struct {
	Name     string
	envVar   string
	value    interface{}
	prev     interface{}
	backup   interface{}
	preset   interface{}
	validate func(string, *PARAMETER) string
	apply    func(*PARAMETER)
	varType  string
}

func (c *CONFIGURATION) NewParameter(name string, preset interface{}) *PARAMETER {
	return c.NewOSEnvParameter(name, "", preset)

}

//Get parameter value from OS environment variable. If not present, use preset
func (c *CONFIGURATION) NewOSEnvParameter(name string, env string, preset interface{}) *PARAMETER {
	if c.Exists(name) {
		return nil
	}

	var t string
	var pre interface{}
	switch vv := preset.(type) {
	case string:
		t = TypeString
		pre = preset.(string)
	case bool:
		t = TypeBool
		pre = preset.(bool)
	case float32:
		t = TypeFloat64
		pre = float64(preset.(float32))
	case float64:
		t = TypeFloat64
		pre = preset.(float64)
	case int:
		t = TypeInt64
		pre = int64(preset.(int))
	case int64:
		t = TypeInt64
		pre = preset
	default:
		//unrecognized default value type
		log.Println("Unrecognized value type:", vv, ", preset:", preset)
		return nil
	}

	var val interface{}
	v := os.Getenv(env)
	if len(v) == 0 {
		val = pre
		log.Tracef("Value for %s from preset %s.", name, val)
	} else {
		log.Tracef("Value for %s from OS env is %s.", name, v)
		var err error = nil
		switch t {
		case TypeString:
			val = v
		case TypeBool:
			val, err = strconv.ParseBool(v)
		case TypeFloat64:
			val, err = strconv.ParseFloat(v, 64)
		case TypeInt64:
			val, err = strconv.ParseInt(v, 10, 64)
		default:
			return nil
		}
		if err != nil {
			return nil
		}
	}

	p := &PARAMETER{
		Name:    name,
		envVar:  env,
		value:   val,
		preset:  pre,
		varType: t,
		validate: func(string, *PARAMETER) string {
			return ""
		},
		apply: func(p *PARAMETER) {
		}}

	c.AddParameter(p)
	return p

}

func (p *PARAMETER) Get() interface{} {
	return p.value
}

func (p *PARAMETER) GetAsString() string {
	v := p.value
	switch p.Type() {
	case TypeBool:
		return strconv.FormatBool(v.(bool))
	case TypeFloat64:
		return strconv.FormatFloat(v.(float64), 'f', -1, 64)
	case TypeInt64:
		return strconv.FormatInt(v.(int64), 10)
	case TypeString:
		return v.(string)
	default:
		log.Error("Impossible happens.")
		return ""
	}
}

func (p *PARAMETER) Set(v string) {
	var val interface{}
	var err error = nil
	switch p.varType {
	case TypeBool:
		val, err = strconv.ParseBool(v)
	case TypeInt64:
		val, err = strconv.ParseInt(v, 10, 64)
	case TypeFloat64:
		val, err = strconv.ParseFloat(v, 64)
	default: //TypeString
		val = v
	}
	if err == nil {
		p.prev = p.value
		p.value = val
	}
}

func (p *PARAMETER) Type() string {
	return p.varType
}

func (p *PARAMETER) RestoreDefault() interface{} {
	p.value = p.preset
	return p.value
}

func (p *PARAMETER) RollBack() interface{} {
	p.value = p.prev
	return p.value
}

func (p *PARAMETER) Backup() {
	p.backup = p.value
}

func (p *PARAMETER) Restore() interface{} {
	p.value = p.backup
	return p.value
}

func (p *PARAMETER) SetValidateFunction(f func(string, *PARAMETER) string) {
	p.validate = f
}

func (p *PARAMETER) SetApplyFunction(f func(*PARAMETER)) {
	p.apply = f
}

func (p *PARAMETER) Validate(v string) string {
	return p.validate(v, p)
}

func (p *PARAMETER) Equal(v string) int {
	var equal int = ERROR
	switch p.varType {
	case TypeBool:
		val, err := strconv.ParseBool(v)
		if err == nil {
			if p.value.(bool) == val {
				equal = EQUAL
			} else {
				equal = UNEQUAL
			}
		}
	case TypeInt64:
		val, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			if p.value.(int64) == val {
				equal = EQUAL
			} else {
				equal = UNEQUAL
			}
		}
	case TypeFloat64:
		val, err := strconv.ParseFloat(v, 64)
		if err == nil {
			if p.value.(float64) == val {
				equal = EQUAL
			} else {
				equal = UNEQUAL
			}
		}
	default: //TypeString
		val := v
		if p.value.(string) == val {
			equal = EQUAL
		} else {
			equal = UNEQUAL
		}
	}
	return equal
}
