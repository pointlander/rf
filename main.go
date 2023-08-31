// Copyright 2023 The RF Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/samuel/go-hackrf/hackrf"
)

var (
	// FlagLearn a point
	FlagLearn = flag.String("learn", "", "learn a point")
	// FlagInfer
	FlagInfer = flag.Bool("infer", false, "inference mode")
)

// Point is a point
type Point struct {
	Name   string
	Points [][]byte
}

// Points is a set of points
type Points map[string]Point

func main() {
	flag.Parse()

	err := hackrf.Init()
	if err != nil {
		panic(err)
	}
	defer hackrf.Exit()
	device, err := hackrf.Open()
	if err != nil {
		panic(err)
	}
	fmt.Println("Device opened")
	device.SetFreq(1e6)
	defer device.Close()

	input, err := os.Open("points.gob")
	points := make(Points)
	if err == nil {
		decoder := gob.NewDecoder(input)
		err = decoder.Decode(&points)
		if err != nil {
			panic(err)
		}
	}
	input.Close()

	if *FlagLearn != "" {
		fmt.Println("wait 5 seconds")
		time.Sleep(5 * time.Second)

		err = device.StartRX(func(buff []byte) error {
			entry := points[*FlagLearn]
			entry.Name = *FlagLearn
			entry.Points = append(entry.Points, buff)
			fmt.Println(len(buff))
			points[*FlagLearn] = entry
			return nil
		})
		if err != nil {
			panic(err)
		}

		time.Sleep(10 * time.Second)
		err = device.StopRX()
		if err != nil {
			panic(err)
		}

		output, err := os.Create("points.gob")
		if err != nil {
			panic(err)
		}
		defer output.Close()
		encoder := gob.NewEncoder(output)
		err = encoder.Encode(points)
		if err != nil {
			panic(err)
		}
		return
	}

	if *FlagInfer {
		err = device.StartRX(func(buff []byte) error {
			name, min := "", math.MaxFloat64
			for _, entry := range points {
				for _, point := range entry.Points {
					sum := 0.0
					for key, value := range buff {
						diff := float64(point[key]) - float64(value)
						sum += diff * diff
					}
					if sum < min {
						min, name = sum, entry.Name
					}
				}
			}
			fmt.Printf("%s %f\n", name, min)
			return nil
		})
		if err != nil {
			panic(err)
		}
		for {
			time.Sleep(1 * time.Second)
		}
	}
}
