package server_test

import (
  . "github.com/onsi/ginkgo"
  . "github.com/onsi/gomega"

  "log"
  "math/rand"
  "testing"
  "time"
)

func TestServer(t *testing.T) {
  RegisterFailHandler(Fail)
  RunSpecs(t, "Server Suite")
}

var _ = BeforeSuite(func() {
  rand.Seed(time.Now().UTC().UnixNano())
  log.Printf("Random seed is set\n\n\n")
})
