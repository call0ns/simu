// truck project main.go
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"simu/util/levlog"

	"github.com/nats-io/nats"
)

type City struct {
	name        string
	distanceMap map[string]int
}

var (
	errNotNeighborCity = errors.New("truck: Not a neighbor city")
	errInitialCity     = errors.New("truck: Initial City not found")
)

func (c *City) distanceFrom(neighbor string) (int, error) {
	if d, ok := c.distanceMap[neighbor]; !ok {
		return 0, errNotNeighborCity
	} else {
		return d, nil
	}
}
func readMap(mapFileName string) map[string]*City {
	mapFile, err := os.Open(mapFileName)
	if mapFile != nil {
		defer mapFile.Close()
	}
	if err != nil {
		levlog.Fatal(err)
	}
	var numCities int
	if _, err = fmt.Fscanln(mapFile, &numCities); err != nil {
		levlog.Fatal(err)
	}
	cityNetwork := make(map[string]*City, numCities)
	var cityName, neighborName string
	var numLinks, distance int
	for i := 0; i < numCities; i++ {
		if _, err = fmt.Fscanln(mapFile, &cityName, &numLinks); err != nil {
			levlog.Fatal(err)
		}
		distanceMap := make(map[string]int, numLinks)
		for j := 0; j < numLinks; j++ {
			if _, err = fmt.Fscanln(mapFile, &neighborName, &distance); err != nil {
				levlog.Fatal(err)
			}
			distanceMap[neighborName] = distance
		}
		cityNetwork[cityName] = &City{cityName, distanceMap}
	}
	return cityNetwork
}

type parcel struct {
	id     string
	dest   string
	weight int
}

type status int

const (
	READY = status(iota)
	IDLE
	BUSY
	STOPPED
)

type Truck struct {
	currentCity                          *City
	parcelMap                            map[string]*parcel
	cityNetwork                          map[string]*City
	id                                   string
	totalCap                             int
	remainCap                            int
	speed                                int
	loadDuration                         time.Duration
	workingTime                          time.Duration
	initialCityName                      string
	ncOut                                *nats.Conn
	ncIn                                 *nats.Conn
	stat                                 status
	StopCh                               chan bool
	lock4Stat, lock4LastCallHappened     sync.Mutex
	lock4ParcelMap                       sync.RWMutex
	timeToStop                           int64 //unixnano
	timeToIdle                           int64
	lastCallHappened                     bool
	loadTimes, unloadTimes               int
	driveTime, driveDistance, travelCost int64
	titleReport                          string
	subLoad, subUnload, subList, subGoto *nats.Subscription
}

func (t *Truck) Report() (driveDistance, driveTime, travelCost int64, loadTimes, unloadTimes int) {
	return t.driveDistance, t.driveTime, t.travelCost, t.loadTimes, t.unloadTimes
}

func (t *Truck) PubReport() {
	levlog.Info("Send a report")
	data := fmt.Sprintf("%d\n%d\n", t.driveDistance*int64(t.totalCap), t.travelCost)
	t.ncOut.Publish(t.titleReport, []byte(data))
}

func NewTruck(
	natsServers []string,
	cityNetwork map[string]*City,
	id string,
	capacity, speed int,
	loadDuration, workingTime time.Duration,
	initialCityName string) (*Truck, error) {
	iCity, ok := cityNetwork[initialCityName]
	if !ok {
		return nil, errNotNeighborCity
	}
	opt := nats.DefaultOptions
	opt.Servers = natsServers
	var ncIn, ncOut *nats.Conn
	var err error
	if ncIn, err = opt.Connect(); err != nil {
		return nil, err
	}
	if ncOut, err = opt.Connect(); err != nil {
		ncIn.Close()
		return nil, err
	}
	t := Truck{
		currentCity:     iCity,
		parcelMap:       make(map[string]*parcel, 128),
		cityNetwork:     cityNetwork,
		id:              id,
		totalCap:        capacity,
		remainCap:       capacity,
		speed:           speed,
		loadDuration:    loadDuration,
		workingTime:     workingTime,
		initialCityName: initialCityName,
		ncIn:            ncIn,
		ncOut:           ncOut,
		stat:            READY,
		StopCh:          make(chan bool),
		titleReport:     "Report.Truck." + id,
	}
	if err = t.start(); err != nil {
		return &t, err
	}
	return &t, nil
}
func (t *Truck) Close() {
	if t.ncIn != nil {
		t.ncIn.Close()
	}
	if t.ncOut != nil {
		t.PubReport()
		t.ncOut.Close()
	}
}
func (t *Truck) getStatus() (result int) { //return 0 for idle, 1 for busy, -1 for stopped
	t.lock4Stat.Lock()
	switch t.stat {
	case READY:
		t.stat = IDLE
		t.lock4Stat.Unlock()
		//		go t.arrive(0)
		t.timeToStop = time.Now().UnixNano() + t.workingTime.Nanoseconds()
		ch := time.After(t.workingTime)
		go func(ch <-chan time.Time) {
			<-ch
			t.lock4Stat.Lock()
			t.stat = STOPPED
			t.lock4Stat.Unlock()
			levlog.Info("Work time ended.")
		}(ch)
		return 0
	case IDLE:
		t.lock4Stat.Unlock()
		return 0
	case BUSY:
		t.lock4Stat.Unlock()
		return 1
	default: //stopped
		t.lock4Stat.Unlock()
		return -1
	}
}
func (t *Truck) startBusy() (result int) { //return 1 for success, 0 for busy, -1 for stopped
	t.lock4Stat.Lock()
	switch t.stat {
	case READY:
		t.stat = BUSY
		t.lock4Stat.Unlock()
		//		go t.arrive(0)
		ch := time.After(t.workingTime)
		go func(ch <-chan time.Time) {
			<-ch
			t.lock4Stat.Lock()
			t.stat = STOPPED
			t.lock4Stat.Unlock()
		}(ch)
		return 1
	case IDLE:
		t.stat = BUSY
		t.lock4Stat.Unlock()
		return 1
	case BUSY:
		t.lock4Stat.Unlock()
		return 0
	default: //stopped
		t.lock4Stat.Unlock()
		return -1
	}
}
func (t *Truck) endBusy() {
	t.lock4Stat.Lock()
	if t.stat == BUSY {
		t.stat = IDLE
	}
	t.lock4Stat.Unlock()
}
func (t *Truck) lastCall() bool {
	t.lock4LastCallHappened.Lock()
	if t.lastCallHappened {
		t.lock4LastCallHappened.Unlock()
		return false
	} else {
		t.lastCallHappened = true
		t.lock4LastCallHappened.Unlock()
		return true
	}
}
func (t *Truck) apiLoad() (err error) {
	handle := func(m *nats.Msg) {
		levlog.Info("Receive a load call")
		levlog.Debug("Load call data: ", string(m.Data))
		if m.Reply == "" {
			levlog.Warning("Receive a load call with empty reply title")
			return
		}
		var reply string
		accept := true
		pList := strings.NewReader(string(m.Data))
		var num int
		var tmpParcelMap map[string]*parcel
		var isRepeatedLoad bool
		if _, err := fmt.Fscanln(pList, &num); err != nil { //read data
			levlog.Warning("load call format error")
			reply = "Reject\nFormatError\n"
			accept = false
		} else {
			var pID, pDest string
			var pWeight int
			tmpParcelMap = make(map[string]*parcel, num)
			for i := 0; i < num; i++ {
				if _, err := fmt.Fscanln(pList, &pID, &pDest, &pWeight); err != nil {
					levlog.Warning("load call format error")
					reply = "Reject\nFormatError\n"
					accept = false
					break
				}
				if _, ok := tmpParcelMap[pID]; ok {
					isRepeatedLoad = true
					break
				}
				tmpParcelMap[pID] = &parcel{
					id:     pID,
					dest:   pDest,
					weight: pWeight,
				}
			}
		}
		if accept {
			if res := t.startBusy(); res == 0 { //busy
				reply = "Reject\nBusy\n"
				accept = false
			} else if res < 0 { //stopped
				reply = "Stopped\n"
				accept = false
				if t.lastCall() {
					defer close(t.StopCh)
				}
			} else { //idle
				ch := time.After(t.loadDuration)
				t.timeToIdle = time.Now().UnixNano() + t.loadDuration.Nanoseconds()
				t.loadTimes++
				if isRepeatedLoad {
					reply = "Reject\nRepeatedLoad\n"
					accept = false
				} else {
					var sumWeight int
					alreadyLoadedMap := make(map[string]bool, num)

					t.lock4ParcelMap.RLock()
					for k, v := range tmpParcelMap { //check already loaded
						if _, ok := t.parcelMap[k]; ok {
							accept = false
							alreadyLoadedMap[k] = true
						}
						sumWeight += v.weight
					}
					t.lock4ParcelMap.RUnlock()

					if !accept { //some parcels already loaded
						reply = "Reject\nAlreadyLoaded\n"
						reply += fmt.Sprintln(len(alreadyLoadedMap))
						for k := range alreadyLoadedMap {
							reply += fmt.Sprintln(k)
						}
					}
					t.lock4ParcelMap.Lock()
					if accept && sumWeight > t.remainCap { //check overload
						reply = "Reject\nOverload\n"
						accept = false
					}
					if accept {
						t.remainCap -= sumWeight
						for k, v := range tmpParcelMap {
							t.parcelMap[k] = v
						}
						reply = "Accept\n"
					}
					t.lock4ParcelMap.Unlock()
				}
				<-ch
				t.endBusy()
			}
		}

		if err := t.ncOut.Publish(m.Reply, []byte(reply)); err != nil {
			levlog.Error(err)
		} else {
			levlog.Debug("Reply load to ", m.Reply, " with: ", reply)
		}
	}
	gohandle := func(m *nats.Msg) {
		go handle(m)
	}
	title := fmt.Sprintf("Truck.%s.Load", t.id)
	t.subLoad, err = t.ncIn.Subscribe(title, gohandle)
	return err
}
func (t *Truck) apiUnload() (err error) {
	handle := func(m *nats.Msg) {
		levlog.Info("Receive a unload call")
		levlog.Debug("Unload call data: ", string(m.Data))
		if m.Reply == "" {
			levlog.Warning("Receive a unload call with empty reply title")
			return
		}
		var reply string
		accept := true
		pList := strings.NewReader(string(m.Data))
		var num int
		var tmpParcelMap map[string]bool
		var isRepeatedUnload bool
		if _, err := fmt.Fscanln(pList, &num); err != nil {
			levlog.Warning("unload call format error")
			reply = "Reject\nFormatError\n"
			accept = false
		} else {
			var pID string
			tmpParcelMap = make(map[string]bool, num)
			for i := 0; i < num; i++ {
				if _, err := fmt.Fscanln(pList, &pID); err != nil {
					levlog.Warning("unload call format error")
					reply = "Reject\nFormatError\n"
					accept = false
					break
				}
				if tmpParcelMap[pID] {
					isRepeatedUnload = true
					break
				}
				tmpParcelMap[pID] = true
			}
		}
		if accept {
			if res := t.startBusy(); res == 0 { //busy
				reply = "Reject\nBusy\n"
				accept = false
			} else if res < 0 { //stopped
				reply = "Stopped\n"
				accept = false
				if t.lastCall() {
					defer close(t.StopCh)
				}
			} else { //idle
				ch := time.After(t.loadDuration)
				t.timeToIdle = time.Now().UnixNano() + t.loadDuration.Nanoseconds()
				t.unloadTimes++
				if isRepeatedUnload {
					reply = "Reject\nRepeatedUnload\n"
					accept = false
				} else if num == -1 { //unload all parcels
					reply = fmt.Sprintf("Accept\n%d\n", num)
					sumWeight := 0
					t.lock4ParcelMap.Lock()
					for k, p := range t.parcelMap {
						delete(t.parcelMap, k)
						reply += fmt.Sprintln(p.id, p.dest, p.weight)
						sumWeight += p.weight
					}
					t.remainCap += sumWeight
					t.lock4ParcelMap.Unlock()
				} else { //reload parcels in list
					parcelNotFoundMap := make(map[string]bool, num)
					t.lock4ParcelMap.RLock()
					for k := range tmpParcelMap { // check if all parcels can be found
						if _, ok := t.parcelMap[k]; !ok {
							accept = false
							parcelNotFoundMap[k] = true
						}
					}
					t.lock4ParcelMap.RUnlock()
					if !accept { //some parcels not found
						reply = "Reject\nCanNotFindParcel\n"
						reply += fmt.Sprintln(len(parcelNotFoundMap))
						for k := range parcelNotFoundMap {
							reply += fmt.Sprintln(k)
						}
					} else { //accept
						reply = fmt.Sprintf("Accept\n%d\n", num)
						t.lock4ParcelMap.Lock()
						sumWeight := 0
						for k := range tmpParcelMap {
							p := t.parcelMap[k]
							delete(t.parcelMap, k)
							reply += fmt.Sprintln(p.id, p.dest, p.weight)
							sumWeight += p.weight
						}
						t.remainCap += sumWeight
						t.lock4ParcelMap.Unlock()
					}
				}
				<-ch
				t.endBusy()
			}
		}

		if err := t.ncOut.Publish(m.Reply, []byte(reply)); err != nil {
			levlog.Error(err)
		} else {
			levlog.Debug("Reply unload to ", m.Reply, " with: ", reply)
		}
	}
	gohandle := func(m *nats.Msg) {
		go handle(m)
	}
	title := fmt.Sprintf("Truck.%s.Unload", t.id)
	t.subUnload, err = t.ncIn.Subscribe(title, gohandle)
	return err
}
func (t *Truck) apiList() (err error) {
	handle := func(m *nats.Msg) {
		levlog.Info("Receive a list call")
		levlog.Debug("List call data: ", string(m.Data))
		if m.Reply == "" {
			levlog.Warning("Receive a list call with empty reply title")
			return
		}
		var reply string
		stat := t.getStatus()
		switch stat {
		case 0:
			reply += "Idle\n"
		case 1:
			reply += "Busy\n"
		default:
			reply += "Stopped\n"
			if t.lastCall() {
				defer close(t.StopCh)
			}
		}
		reply += fmt.Sprintln(t.currentCity.name)
		now := time.Now().UnixNano()
		reply += fmt.Sprintln(t.timeToIdle - now) //remain busy time
		reply += fmt.Sprintln(t.timeToStop - now) //remain working time
		reply += fmt.Sprintln(t.loadDuration.Nanoseconds())
		reply += fmt.Sprintln(t.totalCap)
		t.lock4ParcelMap.RLock()
		reply += fmt.Sprintln(t.remainCap)
		reply += fmt.Sprintln(len(t.parcelMap))
		for _, v := range t.parcelMap {
			reply += fmt.Sprintln(v.id, v.dest, v.weight)
		}
		t.lock4ParcelMap.RUnlock()
		if err := t.ncOut.Publish(m.Reply, []byte(reply)); err != nil {
			levlog.Error(err)
		} else {
			levlog.Debug("Reply list to ", m.Reply, " with: ", reply)
		}
	}
	gohandle := func(m *nats.Msg) {
		go handle(m)
	}
	title := fmt.Sprintf("Truck.%s.List", t.id)
	t.subList, err = t.ncIn.Subscribe(title, gohandle)
	return err
}
func (t *Truck) apiGoto() (err error) {
	handle := func(m *nats.Msg) {
		levlog.Info("Receive a goto call")
		levlog.Debug("Goto call data: ", string(m.Data))
		if m.Reply == "" {
			levlog.Warning("Receive a goto call with empty reply title")
			return
		}
		var reply string
		var target string
		city := strings.NewReader(string(m.Data))
		if _, err := fmt.Fscanln(city, &target); err != nil {
			reply = "Reject\nFormatError\n"
			levlog.Warning("goto call format error")
		} else if d, e := t.currentCity.distanceFrom(target); e != nil {
			reply = "Reject\nNoDirectConnection\n"
		} else {
			if res := t.startBusy(); res == 0 {
				reply = "Reject\nBusy\n"
			} else if res < 0 {
				reply = "Stopped\n"
				if t.lastCall() {
					defer close(t.StopCh)
				}
			} else {
				reply = "Accept\n"
				cost := time.Duration(d) * time.Second / time.Duration(t.speed)
				t.driveTime += cost.Nanoseconds()
				t.driveDistance += int64(d)
				t.travelCost += int64(d) * int64(t.totalCap-t.remainCap)
				t.currentCity = t.cityNetwork[target]
				go t.arrive(cost)

			}
		}
		if err := t.ncOut.Publish(m.Reply, []byte(reply)); err != nil {
			levlog.Error(err)
		} else {
			levlog.Debug("Reply goto to ", m.Reply, " with: ", reply)
		}
	}
	gohandle := func(m *nats.Msg) {
		go handle(m)
	}
	title := fmt.Sprintf("Truck.%s.Goto", t.id)
	t.subGoto, err = t.ncIn.Subscribe(title, gohandle)
	return err
}
func (t *Truck) arrive(cost time.Duration) {
	ch := time.After(cost)
	t.timeToIdle = time.Now().UnixNano() + cost.Nanoseconds()
	<-ch
	t.endBusy()
	title := fmt.Sprintf("Arrive.%s", t.currentCity.name)
	data := fmt.Sprintln(t.id)
	if err := t.ncOut.Publish(title, []byte(data)); err != nil {
		levlog.Error(err)
	} else {
		levlog.Info("Send Arrive to ", title, " with: ", data)
	}
}
func (t *Truck) start() (err error) {
	if err = t.apiList(); err != nil {
		return err
	}
	if err = t.apiLoad(); err != nil {
		return err
	}
	if err = t.apiUnload(); err != nil {
		return err
	}
	if err = t.apiGoto(); err != nil {
		return err
	}
	t.arrive(0)
	go func() {
		tc := time.Tick(time.Second * 5)
		for {
			<-tc
			t.PubReport()
		}
	}()
	levlog.Info("Started")
	return
}

func main() {
	var loglevel, capacity, speed, loadDuration, workingTime, timeFactor int
	var trunkID, initialCity, mapFileName, natsServers string

	GetIntWithDefault(&loglevel, "LOG_LEVEL", "log", 4, "log level")
	GetIntWithDefault(&capacity, "CAPACITY", "c", 2000, "capacity")
	GetIntWithDefault(&speed, "SPEED", "s", 1, "speed(m/s)")
	GetIntWithDefault(&loadDuration, "LOAD_DURATION", "ld", 60, "load duration(min)")
	GetIntWithDefault(&workingTime, "WORKING_TIME", "w", 10000, "working time(min)")
	GetIntWithDefault(&timeFactor, "TIME_FACTOR", "f", 1000, "time factor")
	GetStringWithDefault(&trunkID, "TRUCK_ID", "id", "t0", "truck id")
	GetStringWithDefault(&initialCity, "INIT_CITY", "init", "L_A", "initial city")
	GetStringWithDefault(&mapFileName, "MAP_NAME", "map", "map.data", "map file name")
	GetStringWithDefault(&natsServers, "NATS_URI", "nats", nats.DefaultURL, "nats servers, split by ';'")
	Finish()

	levlog.Start(loglevel)
	cityNetwork := readMap(mapFileName)

	natsServers = strings.Replace(natsServers, "tcp", "nats", -1)
	servers := strings.Split(natsServers, ";")

	realWorkTime := time.Minute * time.Duration(workingTime) / time.Duration(timeFactor)
	realLoadDuration := time.Minute * time.Duration(loadDuration) / time.Duration(timeFactor)
	realSpeed := speed * timeFactor
	t, e := NewTruck(
		servers,
		cityNetwork,
		trunkID,
		capacity, realSpeed,
		realLoadDuration, realWorkTime,
		initialCity)
	if e != nil {
		levlog.Fatal(e)
	}
	<-t.StopCh
	t.Close()
	var driveDistance, realDriveTime, travelCost, loadTimes, unloadTimes = t.Report()
	tf := int64(timeFactor)
	driveTime := time.Duration(realDriveTime * tf)
	levlog.Info("drive distance: ", driveDistance)
	levlog.Info("drive time: ", driveTime)
	levlog.Info("load times: ", loadTimes)
	levlog.Info("unload times: ", unloadTimes)
	levlog.Info("sum(distance*load): ", travelCost)
}

func GetIntWithDefault(pval *int, evnName, name string, defaultVal int, usage string) {
	v := os.Getenv(evnName)
	if v == "" {
		flag.IntVar(pval, name, defaultVal, usage)
		return
	}
	val, err := strconv.Atoi(v)
	if err != nil {
		levlog.Error(err)
		val = defaultVal
	}
	flag.IntVar(pval, name, val, usage)
}

func GetStringWithDefault(pval *string, evnName, name, defaultVal string, usage string) {
	val := os.Getenv(evnName)
	if val == "" {
		val = defaultVal
	}
	flag.StringVar(pval, name, val, usage)
}

func Finish() {
	flag.Parse()
}
