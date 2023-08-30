// Copyright 2023 The RF Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/samuel/go-hackrf/hackrf"
)

func main() {
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
	defer device.Close()
}
