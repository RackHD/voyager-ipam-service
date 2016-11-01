package main

import (
  "flag"
  "log"

  "github.com/RackHD/voyager-ipam-service/server"
)

var (
  amqpAddress = flag.String("amqp-address", "amqp://guest:guest@rabbitmq:5672/", "AMQP URI")
  ipamAddress = flag.String("ipam-address", "ipam:8000", "Address of the IPAM instance to connect to")
)

func init() {
  flag.Parse()
}

func main() {
  s := server.NewServer(*amqpAddress, *ipamAddress)
  if s.MQ == nil {
    log.Fatalf("Could not connect to RabbitMQ")
  }
  defer s.MQ.Close()

  s.Start()

  select {}
}
