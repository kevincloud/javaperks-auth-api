package main

import "github.com/kevincloud/javaperks-auth-api/api"

func main() {
	a := api.New("5825")

	a.Run()
}
