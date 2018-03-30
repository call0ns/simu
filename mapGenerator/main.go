// mapGenerator project main.go
package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"repo.oam.ericloud/paas.git/poc2015/util"
)

type City struct {
	name           string
	numConnRemain  int
	neighborCities map[string]Neighbor
}

type Neighbor struct {
	*City
	distance int
}

func NewCity(name string, numConn int) *City {
	return &City{
		name:           name,
		numConnRemain:  numConn,
		neighborCities: make(map[string]Neighbor, numConn),
	}
}

func (c *City) GetName() string {
	return c.name
}

func (c *City) GetRemainConn() int {
	return c.numConnRemain
}

func (c *City) GetNeighborJSONObj() (ret util.JsonObject) {
	ret = util.NewJsonObject()
	for name, neighborCity := range c.neighborCities {
		ret.PutElement(name, fmt.Sprint(neighborCity.distance))
	}
	return ret
}

func (c *City) String() string {
	ret := fmt.Sprintf("%s %d\n", c.name, len(c.neighborCities))
	for name, neighborCity := range c.neighborCities {
		ret += fmt.Sprintf("\t%s %d\n", name, neighborCity.distance)
	}
	return ret
}

func ConnectCities(a *City, b *City, distance int) {
	a.numConnRemain--
	a.neighborCities[b.name] = Neighbor{b, distance}
	b.numConnRemain--
	b.neighborCities[a.name] = Neighbor{a, distance}
}

var (
	largeNames = []string{"L_A", "L_B", "L_C", "L_D", "L_E", "L_F", "L_G", "L_H", "L_I", "L_J"}
	smallNames = []string{"S_a", "S_b", "S_c", "S_d", "S_e", "S_f", "S_g", "S_h", "S_i", "S_j"}
)

const (
	minConnLarge = 3
	maxConnLarge = 7
	minConnSmall = 1
	maxConnSmall = 5
)

func randIntMinMax(min, max int) int {
	return rand.Intn(max-min+1) + min
}

func randGaussion64(mean, sigma float64) float64 {
	return mean + rand.NormFloat64()*sigma
}
func _isConnected(c *City, ConnectedSet map[string]bool) {
	ConnectedSet[c.name] = true
	for name, neighborCity := range c.neighborCities {
		if _, ok := ConnectedSet[name]; !ok {
			_isConnected(neighborCity.City, ConnectedSet)
		}
	}
}
func isConnected(listCity []*City) bool {
	l := len(listCity)
	if l == 0 {
		return true
	}
	ConnectedSet := make(map[string]bool, l)
	_isConnected(listCity[0], ConnectedSet)
	if l == len(ConnectedSet) {
		return true
	} else {
		return false
	}
}
func main() {
	var numLarge, numSmall, avgDist, stdDist int
	flag.IntVar(&numLarge, "l", 10, "number of large cities")
	flag.IntVar(&numSmall, "s", 10, "number of small cities")
	flag.IntVar(&avgDist, "ad", 1000, "average distance between each pair of neighbors(m)")
	flag.IntVar(&stdDist, "sd", 300, "standard deviation distance between each pair of neighbors(m)")
	flag.Parse()
	numTotal := numLarge + numSmall
	rand.Seed(time.Now().Unix())
	remainCity := make([]*City, 0, numTotal)
	listCity := make([]*City, 0, numTotal)

	//new large cities
	l := len(largeNames)
	var c *City
	for i := 0; i < numLarge; i++ {
		num := randIntMinMax(minConnLarge, maxConnLarge)
		if i < l {
			c = NewCity(largeNames[i], num)
		} else {
			c = NewCity(fmt.Sprintf("L_%d", i), num)
		}
		remainCity = append(remainCity, c)
		listCity = append(listCity, c)
	}

	//new small cities
	l = len(smallNames)
	for i := 0; i < numSmall; i++ {
		num := randIntMinMax(minConnSmall, maxConnSmall)
		if i < l {
			c = NewCity(smallNames[i], num)
		} else {
			c = NewCity(fmt.Sprintf("S_%d", i), num)
		}
		remainCity = append(remainCity, c)
		listCity = append(listCity, c)
	}
	//create Conn
	for numRemain := len(remainCity); numRemain >= 2; {
		x := rand.Intn(numRemain)
		y := rand.Intn(numRemain - 1)
		if y >= x {
			y++
			x, y = y, x
		}
		a := remainCity[x]
		b := remainCity[y]
		g := randGaussion64(float64(avgDist), float64(stdDist))
		d := int(g)
		if d <= 0 {
			d = 1
		}
		ConnectCities(a, b, d)
		if a.GetRemainConn() <= 0 {
			numRemain--
			remainCity[x] = remainCity[numRemain]
		}
		if b.GetRemainConn() <= 0 {
			numRemain--
			remainCity[y] = remainCity[numRemain]
		}
		/*
		fmt.Println(a.name, b.name)
		for i := 0; i < numRemain; i++ {
			fmt.Print(remainCity[i].name, " ")
		}
		fmt.Println()
		*/
	}
	//check if the network is an connected map
	if !isConnected(listCity) {
		fmt.Fprintln(os.Stderr, "Warning: the map is not an connected map")
	}
	//outputJSON
	obj := util.NewJsonObject()
	for i := 0; i < numTotal; i++ {
		c := listCity[i]
		obj.PutElement(c.GetName(), c.GetNeighborJSONObj())
	}
	data := util.Json2Bytes(obj)
	if data == nil {
		panic(errors.New("obj can't marshal"))
	}
	//fmt.Println(string(data))

	//outputLines
	fmt.Println(numTotal)
	for i := 0; i < numTotal; i++ {
		fmt.Print(listCity[i])
	}
}
