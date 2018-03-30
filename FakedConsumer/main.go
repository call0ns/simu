// FakedConsumer project main.go
package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"simu/util/levlog"

	"github.com/nats-io/nats"
)

func main() {
	levlog.Start(levlog.LevelInfo)
	conn, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		levlog.Fatal(err)
	}
	levlog.Debug("started")
	packetRec := make([]int, 0)
	largeCityCount := 0
	count := make(map[string]int)
	startTime := time.Duration(time.Now().UnixNano())
	for {
		msg, err := conn.Request("ParcelGenerator.L_A.GetParcelList", nil, time.Second)
		if err != nil {
			levlog.Error(err)
			continue
		}
		lines := strings.Split(string(msg.Data), "\n")
		linenum, _ := strconv.Atoi(lines[0])
		if linenum == -1 {
			break
		}
		if linenum > 0 {
			msg2, err := conn.Request("ParcelCollecter.L_B.PutParcels", msg.Data, time.Second)
			if err != nil {
				levlog.Error(err)
				continue
			}
			fmt.Println(string(msg2.Data))
		} else {
			continue
		}
		nextLine := 1
		packetRec = append(packetRec, linenum)
		levlog.Debug(linenum)
		if msg.Data != nil {
			//			levlog.Info(string(msg.Data))
			for i := 0; i < linenum; i++ {
				ss := strings.Split(lines[nextLine], " ")
				nextLine++
				if ss[0][0] == 'L' {
					largeCityCount++
				}
				count[ss[1]]++
			}
		}
		startTime += time.Millisecond * 100
		time.Sleep(startTime - time.Duration(time.Now().UnixNano()))
	}
	levlog.Debug(count)
	levlog.Debug(packetRec)
}
