// cmdGenerator project main.go
package main

import (
	"fmt"
	"os"

	"simu/util/levlog"
)

func main() {
	levlog.Start(levlog.LevelTrace)

	// read Map
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
	obj := make(map[string](map[string]int))
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

	parcelExex := "ParcelGenerator/ParcelGenerator"
	//cityExec := "sampleCity/sampleCity"
	cityExec := "city/city"
	truckExec := "truck/truck"
	collectorExec := "ParcelCollector/ParcelCollector"
	timeFactor := 60

	parcelMax := 100
	parcelWorkTime := 30
	loadDuration := 1
	speed := 10

	// start Parcel Gen and city
	fmt.Println("# start Parcel generator and city")
	for k := range obj {
		fmt.Printf("%s -city %s -timeFactor %d -bufferMax %d -workTime %d -natsUrl nats://localhost:4222 > logs/pg.%s.log &\n",
			parcelExex,
			k,
			timeFactor,
			parcelMax,
			parcelWorkTime,
			k,
		)
		fmt.Printf("%s -city %s -nats nats://localhost:4222 > logs/city.%s.log &\n",
			cityExec,
			k,
			k,
		)
		fmt.Printf("%s -city %s -natsUrl nats://localhost:4222 > logs/pc.%s.log &\n",
			collectorExec,
			k,
			k,
		)
	}
	fmt.Println("sleep 5")
	fmt.Println("# start trucks")
	fmt.Println("echo Starting Trucks")
	index := 0
	for k := range obj {
		fmt.Printf("%s -init %s -id t%d -w %d -f %d -ld %d -s %d -nats nats://localhost:4222 > logs/t%d.log &\n",
			truckExec,
			k,
			index,
			parcelWorkTime+10,
			timeFactor,
			loadDuration,
			speed,
			index,
		)
		index++
	}

}
