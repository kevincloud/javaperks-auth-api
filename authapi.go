package main

import "github.com/kevincloud/go-vault/api"

func main() {
	a := api.New("5825")

	a.Run()
}
