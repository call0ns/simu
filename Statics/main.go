// Statics project main.go
package main

import (
	"simu/getPara"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats"
	"repo.oam.ericloud/paas.git/poc2015/util/levlog"
)

type truckReport struct {
	distanceCap     int
	distancePayload int
}

type collectorReport struct {
	parcelVolume int
	delieryTime  int
}

type SimuStatic struct {
	natsUri     string
	truckStstic map[string]truckReport
	truckLock   sync.Mutex
	pcStatic    map[string]collectorReport
	pcLock      sync.Mutex
	natsIn      *nats.Conn
	stopSignal  chan bool
}

func (P *SimuStatic) staticTruck() {
	P.natsIn.Subscribe("Report.Truck.*", func(m *nats.Msg) {
		levlog.Trace("Recved truck msg")
		strs := strings.Split(string(m.Data), "\n")
		if len(strs) < 2 {
			levlog.Error("Report Format Error. from", m.Subject, " Data:", string(m.Data))
			return
		}
		num1, err := strconv.Atoi(strs[0])
		if err != nil {
			levlog.Error("Err parsing number:", strs[0])
			return
		}
		num2, err := strconv.Atoi(strs[1])
		if err != nil {
			levlog.Error("Err parsing number:", strs[1])
			return
		}
		P.truckLock.Lock()
		P.truckStstic[m.Subject] = truckReport{
			num1,
			num2,
		}
		P.truckLock.Unlock()
	})
}

func (P *SimuStatic) staticPC() {
	P.natsIn.Subscribe("Report.Collector.*", func(m *nats.Msg) {
		levlog.Trace("recved collector")
		strs := strings.Split(string(m.Data), "\n")
		if len(strs) < 2 {
			levlog.Error("Report Format Error. from", m.Subject, " Data:", string(m.Data))
			return
		}
		num1, err := strconv.Atoi(strs[0])
		if err != nil {
			levlog.Error("Err parsing number:", strs[0])
			return
		}
		num2, err := strconv.Atoi(strs[1])
		if err != nil {
			levlog.Error("Err parsing number:", strs[1])
			return
		}
		P.pcLock.Lock()
		P.pcStatic[m.Subject] = collectorReport{
			num1,
			num2,
		}
		P.pcLock.Unlock()
	})
}

func (P *SimuStatic) peridicalReport() {
	tick := time.NewTicker(time.Second)

	for {
		select {
		case <-tick.C:
			sumDistCap := 0
			sumDistPayload := 0
			sumParcelVolum := 0
			sumDeliTime := 0

			P.pcLock.Lock()
			for _, v := range P.pcStatic {
				sumDeliTime += v.delieryTime
				sumParcelVolum += v.parcelVolume
			}
			P.pcLock.Unlock()
			P.truckLock.Lock()
			for _, v := range P.truckStstic {
				sumDistCap += v.distanceCap
				sumDistPayload += v.distancePayload
			}
			P.truckLock.Unlock()
			levlog.Infof("%15d,%15d,%15d,%15d",
				sumDistCap,
				sumDistPayload,
				sumParcelVolum,
				sumDeliTime,
			)
		case <-P.stopSignal:
			tick.Stop()
			return
		}
	}
}

func main() {
	levlog.Start(levlog.LevelInfo)
	gen := &SimuStatic{
		truckStstic: make(map[string]truckReport),
		pcStatic:    make(map[string]collectorReport),
		stopSignal:  make(chan bool, 0),
	}
	// get parameter from env or commandline
	getPara.GetStringWithDefault(&gen.natsUri, "natsUrl", "nats://localhost:4222", "usage")
	getPara.Finish()
	var err error
	gen.natsIn, err = nats.Connect(gen.natsUri)
	if err != nil {
		levlog.Fatal(err)
	}
	gen.staticPC()
	gen.staticTruck()
	gen.peridicalReport()
	<-gen.stopSignal
}
