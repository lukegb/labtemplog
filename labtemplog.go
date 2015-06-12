package main

import (
	"fmt"
	"github.com/lukegb/temperedgo"
	"log"
	"net/http"
	"sync"
	"time"
)

type tempStatus struct {
	sync.RWMutex

	currentTemp float64
}

var curTemp tempStatus = tempStatus{}

func main() {
	x := make(chan interface{})
	go func() {
		t := new(temperedgo.Tempered)
		t.Init()
		tds, err := t.DeviceList()
		if err != nil {
			panic(err)
		} else if len(tds) == 0 {
			panic("no devices")
		}

		dev := tds[0]
		if err := dev.Open(); err != nil {
			panic(err)
		}

		sensors, err := dev.Sensors()
		if err != nil {
			panic(err)
		} else if len(sensors) == 0 {
			panic("no sensors on device 0")
		}

		sensor := sensors[0]
		if !sensor.TypeMask.IsType(temperedgo.TEMPERED_SENSOR_TYPE_TEMPERATURE) {
			panic("sensor 0 on device 0 is not a temperature sensor!")
		}

		err = dev.Update()
		if err != nil {
			panic(err)
		}

		for failedCount := 0; failedCount < 10; {
			err = dev.Update()
			if err != nil {
				failedCount++
				log.Println(err)
				time.Sleep(1 * time.Minute)
				continue
			}

			temp, err := sensor.Temperature()
			if err != nil {
				failedCount++
				log.Println(err)
				time.Sleep(1 * time.Minute)
				continue
			}

			curTemp.Lock()
			curTemp.currentTemp = temp
			curTemp.Unlock()
			x <- struct{}{}
			log.Println("bonk")

			time.Sleep(30 * time.Second)
		}
		panic("failed too many times, abort")

	}()
	<-x
	go func() { for { <-x } }()

	http.HandleFunc("/temp", func(w http.ResponseWriter, r *http.Request) {
		curTemp.RLock()
		defer curTemp.RUnlock()
		fmt.Fprintf(w, "%f", curTemp.currentTemp)
	})
	log.Println("RUNNING!")
	log.Fatalln(http.ListenAndServe(":55080", nil))
}
