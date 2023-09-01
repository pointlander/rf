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

	"github.com/jfreymuth/pulse"
	"github.com/samuel/go-dsp/dsp"
	"github.com/samuel/go-hackrf/hackrf"
)

var (
	// FlagLearn a point
	FlagLearn = flag.String("learn", "", "learn a point")
	// FlagInfer
	FlagInfer = flag.Bool("infer", false, "inference mode")
	// FlagRadio
	FlagRadio = flag.Bool("radio", false, "radio mode")
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

	if *FlagRadio {
		c, err := pulse.NewClient()
		if err != nil {
			fmt.Println(err)
			return
		}
		defer c.Close()

		err = device.SetFreq(96.7e6)
		if err != nil {
			panic(err)
		}
		demod := dsp.FMDemodFilter{}
		err = device.StartRX(func(buffer []byte) error {
			iq := make([]complex64, len(buffer)/2)
			for i := range iq {
				iq[i] = complex(float32(buffer[2*i])/128.0, float32(buffer[2*i+1])/128.0)
			}
			audio := make([]float32, len(iq))
			demod.Demodulate(iq, audio)
			index := 0
			synth := func(out []float32) (int, error) {
				for i := range out {
					out[i] = audio[index]
					index++
					if index >= len(audio) {
						return i, pulse.EndOfData
					}
				}
				return len(out), nil
			}
			stream, err := c.NewPlayback(pulse.Float32Reader(synth), pulse.PlaybackLatency(.1))
			if err != nil {
				panic(err)
			}

			stream.Start()
			stream.Drain()
			fmt.Println("Underflow:", stream.Underflow())
			if stream.Error() != nil {
				fmt.Println("Error:", stream.Error())
			}
			stream.Close()
			return nil
		})
		if err != nil {
			panic(err)
		}
		for {
			time.Sleep(1 * time.Second)
		}
	}

	err = device.SetFreq(1e6)
	if err != nil {
		panic(err)
	}
	err = device.SetVGAGain(20)
	if err != nil {
		panic(err)
	}
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
