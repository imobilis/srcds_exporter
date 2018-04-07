package collector

import (
	"log"

	"imobilis/go/srcds/connector"
)

func getConnections() map[string]*connector.Connection {
	connections, err := connections.GetConnections()
	if err != nil {
		log.Fatal(err)
	}
	return connections
}
