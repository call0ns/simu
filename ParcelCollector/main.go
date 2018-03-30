// ParcelCollercter project main.go
package main

import (
	"errors"
	"fmt"
	"simu/getPara"
	"hash/crc64"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/apcera/nats"

	"repo.oam.ericloud/paas.git/poc2015/util/levlog"
)

type Parcel struct {
	From     string
	Weight   int
	ParcelID string
	Time     int64
}

type ParcelCollector struct {
	CurrentCity string
	WorkingTime int
	TimeFactor  int64
	natsUri     string
	natsConn    *nats.Conn
	natsOutConn *nats.Conn
	subTitle    string
	stopSignal  chan bool
	mapFile     string
	distMap     map[string]int
	// statics
	numParcel         int
	numErr            int
	sumWeight         int
	sumWeightDistance int
	sumDeliveryTime   int
	// local map
	receivedMap map[string]bool
	hashTable   *crc64.Table
}

func (P *ParcelCollector) getDistance(obj map[string]map[string]int) {
	distmap := make(map[string]int)
	//	prevmap := make(map[string]string)
	visited := make(map[string]bool)
	maxval := 1 << 31
	for k, _ := range obj {
		visited[k] = false
		distmap[k] = maxval
	}
	visited[P.CurrentCity] = true
	distmap[P.CurrentCity] = 0
	for k, v := range obj[P.CurrentCity] {
		distmap[k] = v
		//		prevmap[k] = P.CurrentCity
	}

	for i := 1; i < len(obj); i++ {
		min := 1 << 31
		mincity := ""
		for city, _ := range obj {
			if !visited[city] && distmap[city] < min {
				min = distmap[city]
				mincity = city
			}
		}
		levlog.Debug("Select ", mincity, " ", min)
		visited[mincity] = true
		for k, v := range obj[mincity] {
			if distmap[k] > min+v {
				distmap[k] = min + v
				//				prevmap[k] = mincity
			}
		}
	}
	P.distMap = distmap
	//	P.prevmap = prevmap
}

func (P *ParcelCollector) readMap(mapFileName string) {
	obj := make(map[string](map[string]int))
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
	P.getDistance(obj)
}

// dataformat :name dest weight
// nameFormat : city.timestemp.randID.hashval
// returnVal : parcelID(only random field), distance, error
func (P *ParcelCollector) handleData(data string) (parcel Parcel, err error) {
	levlog.Debug("handle:", data)
	strs := strings.Split(data, " ")
	if len(strs) < 3 {
		levlog.Error("Len not correct:", len(strs))
		err = errors.New("data format error " + fmt.Sprint(strs))
		return
	}
	strsname := strings.Split(strs[0], ".")
	if len(strsname) < 4 {
		levlog.Error("name format error", len(strsname))
		err = errors.New("data erode")
		return
	}
	if w, err1 := strconv.Atoi(strs[2]); err1 != nil {
		err = err1
		return
	} else {
		parcel.Weight = w
	}
	if t, err1 := strconv.ParseInt(strsname[1], 10, 64); err1 != nil {
		err = err1
		return
	} else {
		parcel.Time = time.Now().UnixNano() - t
	}
	parcel.ParcelID = strsname[2]
	parcel.From = strsname[0]

	oriTitle := strings.Join(strsname[:3], ".")
	oriTitle = oriTitle + " " + strings.Join(strs[1:], " ")
	hashval := fmt.Sprintf("%0.16X", crc64.Checksum([]byte(oriTitle), P.hashTable))
	levlog.Debug("HASHVAL: ", hashval)
	if hashval != strsname[3] {
		levlog.Error("hash Val not correct, ", strsname[3], "----", hashval)
		err = errors.New("Data not Correct")
		return
	}
	// data integrety check finished
	if strs[1] != P.CurrentCity {
		levlog.Error("wrong delivery city:", strs[1], "----", P.CurrentCity)
		err = errors.New("Wrong delivery")
		return
	}
	if recved, ok := P.receivedMap[data]; ok && recved {
		levlog.Error("parcel already stored")
		err = errors.New("Redundent put")
		return
	}
	return
}

func (P *ParcelCollector) Subscribe() {
	var err error
	_, err = P.natsConn.Subscribe(P.subTitle, func(m *nats.Msg) {
		if m.Reply == "" {
			return
		}
		strs := strings.Split(string(m.Data), "\n")
		if len(strs) <= 0 {
			P.natsOutConn.Publish(m.Reply, []byte("Reject\nFormat not correct"))
			return
		}
		num, err := strconv.Atoi(strs[0])
		if err != nil {
			P.natsOutConn.Publish(m.Reply, []byte("Reject\n"+err.Error()))
			return
		}
		if len(strs) < num {
			P.natsOutConn.Publish(m.Reply, []byte("Reject\nnot enough data"))
			return
		}
		rollBackSlice := make([]string, 0)
		sumWeigt := 0
		sumWeigtDistance := 0
		sumDeliveryTime := 0
		for i := 1; i <= num; i++ {
			if parcel, err := P.handleData(strs[i]); err == nil {
				rollBackSlice = append(rollBackSlice, strs[i])
				P.receivedMap[strs[i]] = true
				sumWeigt += parcel.Weight
				sumWeigtDistance += parcel.Weight * P.distMap[parcel.From]
				sumDeliveryTime += int(parcel.Time)
			} else {
				for _, n := range rollBackSlice {
					delete(P.receivedMap, n)
				}
				P.natsOutConn.Publish(m.Reply, []byte("Rejected\n"+err.Error()))
				return
			}
		}
		P.numParcel += num
		P.sumWeight += sumWeigt
		P.sumWeightDistance += sumWeigtDistance
		P.sumDeliveryTime += sumDeliveryTime
		P.natsOutConn.Publish(m.Reply, []byte("Accept"))
	})
	if err != nil {
		levlog.Fatal(err)
	}
}

func (P *ParcelCollector) ReportStatics() {
	levlog.Info("Statics: Parcels-----SumWeight---Sum Weight*Dist---Sum Delivery Time")
	t := time.NewTicker(time.Second * 3)
	for {
		select {
		case <-t.C:
			levlog.Infof("STATIC %s:%10d,%10d,%10d,%15d",
				P.CurrentCity,
				P.numParcel,
				P.sumWeight,
				P.sumWeightDistance,
				P.sumDeliveryTime,
			)
			err := P.natsOutConn.Publish(
				fmt.Sprintf("Report.Collector.%s", P.CurrentCity),
				[]byte(fmt.Sprintf("%d\n%d", P.sumWeightDistance, P.sumDeliveryTime)),
			)
			if err != nil {
				levlog.Error(err)
			}
		case <-P.stopSignal:
			t.Stop()
			levlog.Trace("Leaving Statics")
			return
		}

	}
}

func main() {
	levlog.Start(levlog.LevelInfo)
	gen := &ParcelCollector{
		stopSignal:  make(chan bool, 0),
		numParcel:   0,
		receivedMap: make(map[string]bool),
		hashTable:   crc64.MakeTable(0xC96C5795D7870F42),
	}
	var TimeFactor int
	var mapFile string
	// get parameter from env or commandline
	getPara.GetStringWithDefault(&gen.CurrentCity, "city", "L_B", "city name")
	getPara.GetIntWithDefault(&gen.WorkingTime, "workTime", 20, "work time")
	getPara.GetIntWithDefault(&TimeFactor, "timeFactor", 60, "time Factor")
	getPara.GetStringWithDefault(&gen.natsUri, "natsUrl", "nats://localhost:4222", "nats uri")
	getPara.GetStringWithDefault(&mapFile, "mapFile", "map.data", "map file name")
	getPara.Finish()

	gen.subTitle = fmt.Sprintf("ParcelCollecter.%s.PutParcels", gen.CurrentCity)
	gen.TimeFactor = int64(time.Minute) / int64(TimeFactor)

	conn, err := nats.Connect(gen.natsUri)
	gen.natsConn = conn
	if err != nil {
		levlog.Fatal(err)
	}
	conn, err = nats.Connect(gen.natsUri)
	gen.natsOutConn = conn
	if err != nil {
		levlog.Fatal(err)
	}
	gen.readMap(mapFile)
	gen.Subscribe()
	levlog.Infof("Parcel Collector: %s waiting for API call...", gen.CurrentCity)
	go gen.ReportStatics()
	<-gen.stopSignal

}
