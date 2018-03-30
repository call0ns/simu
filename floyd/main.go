// floyd project main.go
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"simu/util/levlog"
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
		return 0x7FFFFFFF, errNotNeighborCity
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

func main() {
	var mapFileName string
	GetStringWithDefault(&mapFileName, "MAP_NAME", "map", "map.data", "map file name")
	Finish()
	cityNetwork := readMap(mapFileName)
	for _, c := range cityNetwork {
		for _, a := range cityNetwork {
			for _, b := range cityNetwork {
				if a == b || a == c || b == c {
					continue
				}
				dAC, eAC := a.distanceFrom(c.name)
				dAB, _ := a.distanceFrom(b.name)
				dCB, eCB := c.distanceFrom(b.name)
				dACB := dAC + dCB
				if eAC == nil && eCB == nil && (dACB) < dAB {
					a.distanceMap[b.name] = dACB
				}
			}
		}
	}
	fmt.Println(len(cityNetwork))
	for _, c := range cityNetwork {
		fmt.Println(c.name, len(c.distanceMap))
		for name, dist := range c.distanceMap {
			fmt.Printf("\t%s %d\n", name, dist)
		}
	}
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
