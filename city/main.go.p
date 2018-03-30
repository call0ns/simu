package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"simu/getPara"

	"github.com/nats-io/nats"
	"repo.oam.ericloud/paas.git/poc2015/util/levlog"
)

type Parcel struct {
	Dest     string
	Weight   int
	ParcelID string
	Str      string
}

func (p Parcel) ToByte() []byte {
	return []byte(fmt.Sprintf("%s %s %d\n", p.ParcelID, p.Dest, p.Weight))
}

func toBytes(args ...interface{}) []byte {
	return []byte(fmt.Sprint(args...))
}

type city struct {
	CurrentCity string
	orimap      map[string]map[string]int
	distmap     map[string]int
	prevmap     map[string]string
	nextmap     map[string]string
	natsUri     string
	natsConn    *nats.Conn
	natsConnOut *nats.Conn
	natsTimeOut time.Duration
	capacity    int

	storedParcel    map[string][]Parcel
	storedParcelMux sync.Mutex
	collected       map[string]bool

	startSignal chan bool
}

type listParam struct {
	status         string
	city           string
	busyTime       int
	remainTime     int
	loadDuration   int
	capacity       int
	remainCapacity int
	numberParcels  int
	parcels        []Parcel
}

func parseInt(str string) int {
	i, e := strconv.Atoi(str)
	if e != nil {
		levlog.Error(e)
		i = 0
	}
	return i
}

func getParcel(data string) Parcel {
	strs := strings.Split(data, " ")
	return Parcel{
		ParcelID: strs[0],
		Dest:     strs[1],
		Weight:   parseInt(strs[2]),
		Str:      data,
	}
}

func (P *city) listParam(truckID string) *listParam {
	levlog.Trace("List truck: ", truckID)
	//levlog.Trace([]byte(truckID))
	msg, err := P.natsConnOut.Request(fmt.Sprintf("Truck.%s.List", truckID), []byte("asd"), time.Second)
	if err != nil {
		levlog.Error(err)
		return nil
	}
	strs := strings.Split(string(msg.Data), "\n")
	if len(strs) < 8 {
		levlog.Error("List Not Correct")
	}
	ret := listParam{
		status:         strs[0],
		city:           strs[1],
		busyTime:       parseInt(strs[2]),
		remainTime:     parseInt(strs[3]),
		loadDuration:   parseInt(strs[4]),
		capacity:       parseInt(strs[5]),
		remainCapacity: parseInt(strs[6]),
		numberParcels:  parseInt(strs[7]),
	}
	ret.parcels = make([]Parcel, ret.numberParcels)
	for i := 0; i < ret.numberParcels; i++ {
		ret.parcels[i] = getParcel(strs[i+8])
	}
	P.natsTimeOut = time.Duration(ret.loadDuration) + time.Second
	P.capacity = ret.capacity
	return &ret
}

func (P *city) getMap() {
	obj := make(map[string](map[string]int))
	mapFile, err := os.Open("map.data")
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
		obj[cityName] = distanceMap
	}
	P.orimap = obj

	P.distmap = make(map[string]int)
	P.prevmap = make(map[string]string)
	P.nextmap = make(map[string]string)
	visited := make(map[string]bool)
	maxval := 1 << 31
	for k, _ := range P.orimap {
		visited[k] = false
		P.distmap[k] = maxval
	}
	visited[P.CurrentCity] = true
	P.distmap[P.CurrentCity] = 0
	for k, v := range P.orimap[P.CurrentCity] {
		P.distmap[k] = v
		P.prevmap[k] = P.CurrentCity
	}

	for i := 1; i < len(P.orimap); i++ {
		min := 1 << 31
		mincity := ""
		for city, _ := range P.orimap {
			if !visited[city] && P.distmap[city] < min {
				min = P.distmap[city]
				mincity = city
			}
		}
		levlog.Debug("Select ", mincity, " ", min)
		visited[mincity] = true
		for k, v := range P.orimap[mincity] {
			if P.distmap[k] > min+v {
				P.distmap[k] = min + v
				P.prevmap[k] = mincity
			}
		}
	}
	levlog.Debug(len(P.prevmap))
	for k, v := range P.prevmap {
		levlog.Debug(k, " ", v)
	}

	P.nextmap[P.CurrentCity] = P.CurrentCity
	for len(P.nextmap) < len(P.distmap) {
		levlog.Debug(len(P.nextmap), " ", len(P.distmap))
		for n, p := range P.prevmap {
			if p == P.CurrentCity {
				P.nextmap[n] = n
			} else if _, ok := P.nextmap[p]; ok {
				P.nextmap[n] = P.nextmap[p]
			}
		}
	}
	for k, v := range P.nextmap {
		levlog.Debug(k, "< ", v)
	}
}

func (P *city) unload(truckID string, parcels []Parcel) bool {
	sendBuffer := make([]byte, 0, 1024*8)
	sendBuffer = append(sendBuffer, []byte(fmt.Sprintf("%d\n", len(parcels)))...)
	for _, v := range parcels {
		sendBuffer = append(sendBuffer, []byte(v.ParcelID+"\n")...)
	}
	msg, err := P.natsConn.Request(fmt.Sprintf("Truck.%s.Unload", truckID), sendBuffer, P.natsTimeOut)
	if err != nil {
		levlog.Error(err)
		return false
	}
	if msg.Data[0] != 'A' {
		levlog.Error(string(msg.Data))
		levlog.Error(string(sendBuffer))
		return false
	}
	return true
}

func (P *city) addParcels(parcels []Parcel) {
	//levlog.Trace("CITY_", P.CurrentCity, "addparcels")
	P.storedParcelMux.Lock()
	for _, v := range parcels {
		nextCity := P.nextmap[v.Dest]
		if P.storedParcel[nextCity] == nil {
			P.storedParcel[nextCity] = make([]Parcel, 0)
		}
		P.storedParcel[nextCity] = append(P.storedParcel[nextCity], v)
	}
	P.storedParcelMux.Unlock()
}

func (P *city) collectParcel() bool {
	if P.storedParcel[P.CurrentCity] == nil {
		return true
	}
	P.storedParcelMux.Lock()
	defer P.storedParcelMux.Unlock()
	sendBuff := make([]byte, 0, 1024*8)
	tosend := make([]Parcel, 0, 0)
	for _, v := range P.storedParcel[P.CurrentCity] {
		if !P.collected[v.ParcelID] {
			tosend = append(tosend, v)
			P.collected[v.ParcelID] = true
		}
	}
	if len(tosend) == 0 {
		return true
	}
	sendBuff = append(sendBuff, ([]byte(fmt.Sprintf("%d\n", len(tosend))))...)
	for _, v := range tosend {
		sendBuff = append(sendBuff, v.ToByte()...)
	}
	delete(P.storedParcel, P.CurrentCity)
	title := fmt.Sprintf("ParcelCollecter.%s.PutParcels", P.CurrentCity)
	msg, err := P.natsConn.Request(title, sendBuff, time.Second)
	if err != nil {
		levlog.Error(err)
		return false
	}
	if msg.Data[0] != 'A' {
		levlog.Error(string(msg.Data))
		return false
	}
	return true
}

func (P *city) sumCity(city string) int {
	if city == P.CurrentCity || P.storedParcel[city] == nil {
		return 0
	}
	sum := 0
	for _, v := range P.storedParcel[city] {
		sum += v.Weight
	}
	return sum
}

func (P *city) pickCity() string {
	P.storedParcelMux.Lock()
	defer P.storedParcelMux.Unlock()
	maxVal := 0
	maxcity := ""
	for k, _ := range P.storedParcel {
		sum := P.sumCity(k)
		sumParcels := sum * P.distmap[k]

		sumNeighbor := 0
		msg, err := P.natsConnOut.Request(fmt.Sprintf("Parcel.%s.List", k), []byte(P.CurrentCity), time.Second)
		if err != nil {
			levlog.Error(err)
		} else {
			strs := strings.Split(string(msg.Data), "\n")
			sumNeighbor += parseInt(strs[0])
		}

		fmt.Println(k, sumParcels, sumNeighbor, P.distmap[k])
		sumParcels += sumNeighbor * 2 / 5

		if sumParcels > maxVal {
			maxVal = sumParcels
			maxcity = k
		}
	}
	if maxcity == "" {
		for k, v := range P.nextmap {
			if k != P.CurrentCity {
				return v
			}
		}
	}
	return maxcity
}

func (P *city) loadParcels(truckID, city string) bool {
	index := 0
	sum := 0
	parcels := P.storedParcel[city]
	for ; index < len(parcels); index++ {
		sum += parcels[index].Weight
		if sum > P.capacity {
			break
		}
	}
	sendBuff := make([]byte, 0, 1024*8)
	sendBuff = append(sendBuff, toBytes(fmt.Sprintf("%d\n", index))...)
	for i := 0; i < index; i++ {
		sendBuff = append(sendBuff, parcels[i].ToByte()...)
	}
	title := fmt.Sprintf("Truck.%s.Load", truckID)
	msg, err := P.natsConn.Request(title, sendBuff, P.natsTimeOut)
	if err != nil {
		levlog.Error(err)
		return false
	}
	if msg.Data[0] != 'A' {
		levlog.Error(string(msg.Data))
		return false
	}
	P.storedParcel[city] = P.storedParcel[city][index:]
	return true
}

func (P *city) handleArrive() {
	P.natsConn.Subscribe(fmt.Sprintf("Arrive.%s", P.CurrentCity), func(msg *nats.Msg) {
		<-P.startSignal
		truckID := strings.Split(string(msg.Data), "\n")[0]
		param := P.listParam(truckID)
		if param == nil || param.status != "Idle" {
			return
		}
		//levlog.Trace(param)
		if !P.unload(truckID, param.parcels) {
			levlog.Error("Error unload")
			return
		}
		P.addParcels(param.parcels)
		if !P.collectParcel() {
			return
		}
		city := P.pickCity()
		if !P.loadParcels(truckID, city) {
			return
		}
		if !P.truckGoto(truckID, city) {
			return
		}
	})
}

func (P *city) reportParcel() (err error) {
	handle := func(m *nats.Msg) {
		levlog.Debug("Receive a list call from: ", string(m.Data))
		if m.Reply == "" {
			levlog.Warning("Receive a list call with empty reply title")
			return
		}
		var reply string
		maxVal := 0
		for k, _ := range P.storedParcel {
			sum := P.sumCity(k)
			sumParcels := sum * P.distmap[k]
			maxVal += sumParcels
		}
		reply += fmt.Sprintln(maxVal)

		if err := P.natsConnOut.Publish(m.Reply, []byte(reply)); err != nil {
			levlog.Error(err)
		} else {
			levlog.Debug("Reply list to ", m.Reply, " with: ", reply)
		}
	}
	gohandle := func(m *nats.Msg) {
		go handle(m)
	}
	title := fmt.Sprintf("Parcel.%s.List", P.CurrentCity)
	P.natsConn.Subscribe(title, gohandle)
	return err
}

func (P *city) truckGoto(truckID, city string) bool {
	levlog.Infof("GOTO %5s:%5s---->%5s", truckID, P.CurrentCity, city)
	msg, err := P.natsConn.Request(fmt.Sprintf("Truck.%s.Goto", truckID), toBytes(city), time.Second)
	if err != nil {
		levlog.Error(err)
		return false
	}
	if msg.Data[0] != 'A' {
		levlog.Error(string(msg.Data))
		return false
	}
	return true
}

func (P *city) fetchParcels() {
	P.startSignal <- true
	close(P.startSignal)
	startTime := time.Duration(time.Now().UnixNano())
	for {
		msg, err := P.natsConn.Request(fmt.Sprintf("ParcelGenerator.%s.GetParcelList", P.CurrentCity), nil, time.Second)
		if err != nil {
			levlog.Error(err)
			continue
		}
		lines := strings.Split(string(msg.Data), "\n")
		linenum, _ := strconv.Atoi(lines[0])
		if linenum == -1 {
			break
		}
		nextLine := 1
		//levlog.Trace(linenum)
		if msg.Data != nil {
			for i := 0; i < linenum; i++ {
				P.addParcels([]Parcel{getParcel(lines[nextLine])})
				nextLine++
			}
		}
		startTime += time.Millisecond * 10
		time.Sleep(startTime - time.Duration(time.Now().UnixNano()))
	}
}

func main() {
	levlog.Start(levlog.LevelDebug)

	P := city{
		startSignal:  make(chan bool),
		storedParcel: make(map[string][]Parcel),
		collected:    make(map[string]bool),
	}

	getPara.GetStringWithDefault(&P.CurrentCity, "city", "L_A", "")
	getPara.GetStringWithDefault(&P.natsUri, "nats", "nats://localhost:4222", "")
	getPara.Finish()

	var err error
	P.natsConn, err = nats.Connect(P.natsUri)
	if err != nil {
		levlog.Fatal(err)
	}
	P.natsConnOut, err = nats.Connect(P.natsUri)
	if err != nil {
		levlog.Fatal(err)
	}
	levlog.Info("City:", P.CurrentCity)
	P.getMap()
	P.handleArrive()
	P.reportParcel()
	go P.fetchParcels()
	for {
		time.Sleep(time.Second)
	}
}
