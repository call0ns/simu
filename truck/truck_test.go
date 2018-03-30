package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats"
	"repo.oam.ericloud/paas.git/poc2015/util/levlog"
)

func Test_truck(test *testing.T) {
	levlog.Start(1)
	nc, err := nats.Connect(nats.DefaultURL)
	defer nc.Close()
	if err != nil {
		test.Error(err)
		return
	}
	var arriveA, arriveB bool
	nc.Subscribe("Arrive.L_A", func(m *nats.Msg) {
		fmt.Println(time.Now())
		fmt.Println("[Arrive]", time.Now(), "Arrive L_A:", string(m.Data))
		arriveA = true
	})
	nc.Subscribe("Arrive.L_B", func(m *nats.Msg) {
		fmt.Println(time.Now())
		fmt.Println("[Arrive]", time.Now(), "Arrive L_B: ", string(m.Data))
		arriveB = true
	})
	cityNetwork := readMap("testMap.data")
	truck, err := NewTruck(
		[]string{nats.DefaultURL},
		cityNetwork,
		"t0",
		1000, 1,
		time.Second, time.Second*20,
		"L_A")
	if err != nil {
		test.Error(err)
		return
	}
	for !arriveA {
		time.Sleep(time.Millisecond * 10)
	}
	var counter int
	testTruck := func(api, data string) {
		i := counter
		counter++
		start := time.Now().UnixNano()
		fmt.Println("[Send]", "num:", i, "now:", time.Now(), api, ":", data)
		m, err := nc.Request("Truck.t0."+api, []byte(data), time.Second*2)
		delay := time.Duration(time.Now().UnixNano() - start)
		if err != nil {
			test.Error(err)
			fmt.Println("[Erro]", "num:", i, "delay:", delay, api, ":", err)
		} else {
			fmt.Println("[Recv]", "num:", i, "delay:", delay, api, ":", string(m.Data))
		}
	}

	fmt.Println("----list: first api call----")
	testTruck("List", "")

	fmt.Println("----load: format error----")
	testTruck("Load", "2\np0 L_B 10\n")

	fmt.Println("----load: repeated load----")
	testTruck("Load", "2\np0 L_B 10\np0 L_B 10\n")

	fmt.Println("----load: overload----")
	testTruck("Load", "1\np0 L_B 1000000\n")

	fmt.Println("----load: accept----")
	testTruck("Load", "1\np0 L_B 10\n")

	go func() {
		time.Sleep(time.Millisecond * 10)
		fmt.Println("----load: busy----")
		testTruck("Load", "1\np0 L_B 10\n")
	}()

	fmt.Println("----load: already loaded----")
	testTruck("Load", "1\np0 L_B 10\n")

	fmt.Println("----goto: no direct link----")
	testTruck("Goto", "L_A")
	fmt.Println("----goto: accept----")
	testTruck("Goto", "L_B")
	fmt.Println("----goto: busy----")
	testTruck("Goto", "L_A")
	fmt.Println("----wait for Arrive L_B----")
	for !arriveB {
		time.Sleep(time.Millisecond * 10)
	}

	fmt.Println("----list----")
	testTruck("List", "")

	fmt.Println("----unload: format error----")
	testTruck("Unload", "2\np0\n")

	fmt.Println("----unload: repeated unload----")
	testTruck("Unload", "2\np0\np0\n")

	fmt.Println("----unload: accept----")
	testTruck("Unload", "1\np0\n")

	go func() {
		fmt.Println("----unload: can not find parcel----")
		testTruck("Unload", "1\np0\n")
	}()

	go func() {
		time.Sleep(time.Millisecond * 10)
		fmt.Println("----unload: busy----")
		testTruck("Unload", "1\np0\n")
	}()

	time.Sleep(time.Millisecond * 100)
	for {
		fmt.Println("----list----")
		testTruck("List", "")
		select {
		case <-truck.StopCh:
			truck.Close()
			fmt.Println("----Truck Report----")
			fmt.Println("driveDistance, driveTime, sum(distance*load), loadTimes, unloadTimes")
			fmt.Println(truck.Report())
			return
		default:
			time.Sleep(5 * time.Second)
		}
	}
}
