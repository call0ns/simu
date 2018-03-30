// Parcel_Generator project main.go
package main

import (
	"fmt"
	"hash/crc64"
	"math/rand"
	"os"
	"simu/getPara"
	"sync"
	"time"

	"simu/util/levlog"

	"github.com/nats-io/nats"
)

type Parcel struct {
	Dest     string
	Weight   int
	ParcelID string
}

type ParcelGenerator struct {
	CurrentCity      string
	RandomSeed       int
	AverageWeight    int
	Sigma            float64
	Rate             int
	BufferMax        int
	WorkingTime      int
	AdjacentCity     []string
	OtherCity        []string
	Parcels          []Parcel
	TimeFactor       int64
	AdjecentCityRate int // in persentage
	buferMux         sync.Mutex
	natsUri          string
	natsConn         *nats.Conn
	subTitle         string
	startTimer       chan bool
	stopSignal       chan bool
	stoped           bool
	mapFile          string
	distMap          map[string]int
	// statics
	numParcel        int
	numDiscarded     int
	numFetched       int
	sumWeight        int
	sumFetchedWeight int
	sumDisWeight     int
	hashTable        *crc64.Table
}

func (P *ParcelGenerator) readMap(mapFileName string) {
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
	// browse the map
	bigCitys := make(map[string]bool)
	adjacentCity := make(map[string]bool)
	otherCity := make(map[string]bool)
	for key := range obj {
		sobj := obj[key]
		if key[0] == 'L' {
			bigCitys[key] = true
		}
		if key == P.CurrentCity {
			for k := range sobj {
				adjacentCity[k] = true
			}
		} else {
			if _, succ := sobj[P.CurrentCity]; succ {
				adjacentCity[key] = true
			}
		}
	}
	for key := range obj {
		if _, ok := adjacentCity[key]; !ok && key != P.CurrentCity {
			otherCity[key] = true
		}
	}
	for key, val := range obj {
		levlog.Debug(key)
		levlog.Debug("\t\t", val)
	}
	P.AdjacentCity = make([]string, 0)
	for k := range adjacentCity {
		P.AdjacentCity = append(P.AdjacentCity, k)
		if _, ok := bigCitys[k]; ok {
			P.AdjacentCity = append(P.AdjacentCity, k)
			P.AdjacentCity = append(P.AdjacentCity, k)
		}
	}
	levlog.Debug(otherCity, " ", len(otherCity))
	if len(otherCity) == 0 {
		P.OtherCity = P.AdjacentCity
	} else {
		P.OtherCity = make([]string, 0)
	}
	for k := range otherCity {
		if k == P.CurrentCity {
			continue
		}
		P.OtherCity = append(P.OtherCity, k)
		if _, ok := bigCitys[k]; ok {
			P.OtherCity = append(P.OtherCity, k)
			P.OtherCity = append(P.OtherCity, k)
		}
	}
	levlog.Debug(P.OtherCity)
	levlog.Debug(P.AdjacentCity)
}

func (P *ParcelGenerator) Simulate() {
	var sumTime float64
	sumTime = 0
	var city = ""
	var weight = 0
	P.startTimer <- true
	StartTime := time.Now().UnixNano()
	close(P.startTimer)
	go P.periodicalStatic()

	for int(sumTime) < P.WorkingTime {
		P.numParcel++
		period := rand.ExpFloat64() / float64(P.Rate)
		sumTime += period
		timeExp := int64(sumTime*float64(P.TimeFactor)) + StartTime
		time.Sleep(time.Duration(timeExp - time.Now().UnixNano()))
		if int(rand.Int31n(100)) < P.AdjecentCityRate {
			city = P.AdjacentCity[rand.Int31n(int32(len(P.AdjacentCity)))]
		} else {
			city = P.OtherCity[rand.Int31n(int32(len(P.OtherCity)))]
		}
		weight = 0
		for weight < 1 {
			weight = int(rand.NormFloat64()*P.Sigma) + P.AverageWeight
		}
		// name : city.timestemp.randID.hashval
		parcelName := fmt.Sprintf("%s.%d.%0.16X", P.CurrentCity, time.Now().UnixNano(), rand.Int())
		hashval := crc64.Checksum([]byte(fmt.Sprintf("%s %s %d", parcelName, city, weight)), P.hashTable)
		parcelName = fmt.Sprintf("%s.%0.16X", parcelName, hashval)
		//		fmt.Println(parcelName)
		levlog.Trace(parcelName, " ", city)
		if len(P.Parcels) < P.BufferMax {
			P.buferMux.Lock()
			P.sumWeight += weight
			P.Parcels = append(P.Parcels, Parcel{city, weight, parcelName})
			P.buferMux.Unlock()
		} else {
			P.sumDisWeight += weight
			P.numDiscarded++
			levlog.Debugf("Discard Parcel: %s weight: %d", parcelName, weight)
		}
	}
}

func (P *ParcelGenerator) Subscribe() {
	var err error
	P.natsConn, err = nats.Connect(P.natsUri)
	if err != nil {
		levlog.Fatal(err)
	}
	cityCount := make(map[string]int)
	cityWcount := make(map[string]int)
	_, err = P.natsConn.Subscribe(P.subTitle, func(m *nats.Msg) {
		<-P.startTimer
		P.buferMux.Lock()
		if P.stoped {
			P.natsConn.Publish(m.Reply, []byte("-1\n"))
			close(P.stopSignal)
			P.natsConn.Close()
			P.buferMux.Unlock()
			levlog.Info("CityCount")
			levlog.Info(cityCount)
			levlog.Info("CityW Cout")
			levlog.Info(cityWcount)
			return
		}
		body := make([]byte, 0, 1024*8)
		body = append(body, ([]byte(fmt.Sprintf("%d\n", len(P.Parcels))))...)
		for _, key := range P.Parcels {
			body = append(body, ([]byte(fmt.Sprintf("%s %s %d\n",
				key.ParcelID,
				key.Dest,
				key.Weight)))...)
			P.sumFetchedWeight += key.Weight
			P.numFetched++
			cityCount[key.Dest]++
			cityWcount[key.Dest] += key.Weight
		}
		body = append(body, ([]byte("\n"))...)
		levlog.Debugf("Get Parcel, %d", len(P.Parcels))
		P.Parcels = P.Parcels[0:0]
		P.buferMux.Unlock()
		err := P.natsConn.Publish(m.Reply, body)
		if err != nil {
			levlog.Error(err)
		}
	})
	if err != nil {
		levlog.Error(err)
	}
}

func (P *ParcelGenerator) periodicalStatic() {
	<-P.startTimer
	tick := time.NewTicker(time.Second)
	prevNumParcel := P.numParcel
	prevWeight := P.sumWeight
	prevWeightDis := P.sumDisWeight
	prevNumParcelDis := P.numDiscarded
	for {
		select {
		case <-P.stopSignal:
			tick.Stop()
			return
		case <-tick.C:
			levlog.Infof("Parcels:%d,discard:%d,fetched:%d  Weight:%d,%d,%d",
				P.numParcel-prevNumParcel,
				P.numDiscarded-prevNumParcelDis,
				P.numParcel-prevNumParcel-(P.numDiscarded-prevNumParcelDis),
				P.sumWeight-prevWeight,
				P.sumDisWeight-prevWeightDis,
				P.sumWeight-prevWeight-(P.sumDisWeight-prevWeightDis),
			)
			prevNumParcel = P.numParcel
			prevWeight = P.sumWeight
			prevWeightDis = P.sumDisWeight
			prevNumParcelDis = P.numDiscarded
		}
	}
}

func main() {
	levlog.Start(levlog.LevelInfo)
	gen := &ParcelGenerator{
		startTimer: make(chan bool, 0),
		stopSignal: make(chan bool, 0),
		stoped:     false,
		numParcel:  0,
		hashTable:  crc64.MakeTable(0xC96C5795D7870F42),
	}
	var TimeFactor int
	var mapFile string
	// get parameter from env or commandline
	getPara.GetStringWithDefault(&gen.CurrentCity, "city", "L_A", "usage")
	getPara.GetIntWithDefault(&gen.RandomSeed, "seed", 12345, "usage")
	getPara.GetIntWithDefault(&gen.AverageWeight, "avgWeight", 15, "usage")
	getPara.GetFloatWithDefault(&gen.Sigma, "sigma", 5, "usage")
	getPara.GetIntWithDefault(&gen.Rate, "rate", 20, "usage")
	getPara.GetIntWithDefault(&gen.BufferMax, "bufferMax", 4000, "usage")
	getPara.GetIntWithDefault(&gen.WorkingTime, "workTime", 20, "usage")
	getPara.GetIntWithDefault(&TimeFactor, "timeFactor", 60, "usage")
	getPara.GetStringWithDefault(&gen.natsUri, "natsUrl", "nats://localhost:4222", "usage")
	getPara.GetStringWithDefault(&mapFile, "mapFile", "map.data", "usage")
	getPara.Finish()

	gen.subTitle = fmt.Sprintf("ParcelGenerator.%s.GetParcelList", gen.CurrentCity)
	gen.TimeFactor = int64(time.Minute) / int64(TimeFactor)
	gen.AdjecentCityRate = 35

	rand.Seed(int64(gen.RandomSeed))

	gen.readMap(mapFile)
	gen.Subscribe()
	levlog.Infof("Parcel Generator: %s waiting for API call...", gen.CurrentCity)
	gen.Simulate()

	gen.buferMux.Lock()
	gen.stoped = true
	gen.buferMux.Unlock()
	<-gen.stopSignal
	levlog.Info("WorkTime ended")

	levlog.Infof("GENERATOR STATICS %s", gen.CurrentCity)
	levlog.Infof("Parcels:%d,discard:%d,fetched:%d  Weight:%d,%d,%d",
		gen.numParcel,
		gen.numDiscarded,
		gen.numFetched,
		gen.sumWeight,
		gen.sumDisWeight,
		gen.sumFetchedWeight,
	)
	//	levlog.Infof("Avg Weight------%f", float64(gen.sumWeight)/float64(gen.numParcel))
	//	levlog.Info("Parcel Generator ", gen.CurrentCity, " Exit")
}
