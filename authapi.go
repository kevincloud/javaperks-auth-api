package main

import (
	"./api"
)

func main() {
	a := api.New("5825")

	a.Run()
}
