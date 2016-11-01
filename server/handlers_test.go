package server_test

import (
	"encoding/json"
	"log"
	"os"

	"github.com/RackHD/ipam/resources"
	"github.com/RackHD/voyager-ipam-service/server"
	"github.com/RackHD/voyager-utilities/models"
	"github.com/RackHD/voyager-utilities/random"
	samqp "github.com/streadway/amqp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server test", func() {
	Describe("AMQP Message Handling", func() {
		var rabbitMQURL string
		var testExchange string
		var testExchangeType string
		var testQueueName string
		var testConsumerTag string
		var testMessage string
		var testRoutingKey string
		var srv *server.Server

		BeforeEach(func() {
			testExchange = models.IpamExchange
			testExchangeType = models.IpamExchangeType
			testQueueName = random.RandQueue()
			testConsumerTag = models.IpamConsumerTag
			testRoutingKey = models.IpamSendQueue

			rabbitMQURL = os.Getenv("RABBITMQ_URL")
			srv = server.NewServer(rabbitMQURL, "127.0.0.1:8000")
			Expect(srv.MQ).ToNot(BeNil())
		})

		AfterEach(func() {
			srv.MQ.Close()
		})

		Context("When a message comes in to voyager-ipam-service", func() {
			var deliveries <-chan samqp.Delivery
			var replies <-chan samqp.Delivery
			var err error
			var replyTo string

			BeforeEach(func() {
				testExchange = "voyager-ipam-service"
				_, deliveries, err = srv.MQ.Listen(testExchange, testExchangeType, testQueueName, testRoutingKey, testConsumerTag)
				Expect(err).ToNot(HaveOccurred())

				replyTo = random.RandQueue()
				_, replies, err = srv.MQ.Listen(testExchange, testExchangeType, replyTo, replyTo, testConsumerTag)
				Expect(err).ToNot(HaveOccurred())

			})

			Context("When a VALID pool type is requested", func() {
				It("INTEGRATION should handle a POOL message with a VALID CREATE ACTION to the 'voyager-ipam-service' exchange", func() {

					testMessage = `{"type":"pool","action":"create","name":"TEST"}`
					err = srv.MQ.Send(testExchange, testExchangeType, testRoutingKey, testMessage, "", replyTo)
					Expect(err).ToNot(HaveOccurred())

					d := <-deliveries
					d.Ack(false)
					err = srv.ProcessMessage(&d)
					Expect(err).ToNot(HaveOccurred())

					message := <-replies
					message.Ack(false)
					Expect(string(message.Body)).ToNot(ContainSubstring("[ERROR]"))
					var pool resources.PoolV1
					err = json.Unmarshal(message.Body, &pool)
					Expect(err).ToNot(HaveOccurred())

					_, err = srv.IPAM.DeletePool(pool.ID, pool)
					Expect(err).ToNot(HaveOccurred())

				})

				It("INTEGRATION should handle a POOL message with an INVALID ACTION to the 'voyager-ipam-service' exchange", func() {
					testMessage = `{"type":"pool","action":"BAD_ACTION","name":"TEST"}`
					err = srv.MQ.Send(testExchange, testExchangeType, testRoutingKey, testMessage, "", replyTo)
					Expect(err).ToNot(HaveOccurred())

					d := <-deliveries
					d.Ack(false)
					err = srv.ProcessMessage(&d)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("When a VALID subnet type is requested", func() {
				var testSubnet models.IPAMSubnetMsg
				var pool resources.PoolV1

				BeforeEach(func() {
					testSubnet.ObjectType = models.SubnetType
					testSubnet.Action = models.CreateAction
					testSubnet.Start = "192.168.1.10"
					testSubnet.End = "192.168.1.20"
					pool = resources.PoolV1{
						Name: "POOL-FOR-SUBNET-TESTS",
					}
					pool, err = srv.IPAM.CreateShowPool(pool)
					Expect(err).ToNot(HaveOccurred())
					testSubnet.Pool = pool.ID
				})

				AfterEach(func() {
					_, err = srv.IPAM.DeletePool(pool.ID, pool)
					Expect(err).ToNot(HaveOccurred())
				})

				It("INTEGRATION should handle a SUBNET message with a VALID CREATE ACTION", func() {

					subnetMsg, err := json.Marshal(testSubnet)
					Expect(err).ToNot(HaveOccurred())

					err = srv.MQ.Send(testExchange, testExchangeType, testRoutingKey, string(subnetMsg), "", replyTo)
					Expect(err).ToNot(HaveOccurred())

					d := <-deliveries
					d.Ack(false)
					err = srv.ProcessMessage(&d)
					Expect(err).ToNot(HaveOccurred())

					r := <-replies
					r.Ack(false)
					Expect(string(r.Body)).ToNot(ContainSubstring("[ERROR]"))

				})

				It("INTEGRATION should handle a SUBNET message with an INVALID ACTION", func() {
					testSubnet.Action = "BAD_ACTION"

					subnetMsg, err := json.Marshal(testSubnet)
					Expect(err).ToNot(HaveOccurred())

					err = srv.MQ.Send(testExchange, testExchangeType, testRoutingKey, string(subnetMsg), "", replyTo)
					Expect(err).ToNot(HaveOccurred())

					d := <-deliveries
					d.Ack(false)
					err = srv.ProcessMessage(&d)
					Expect(err).To(HaveOccurred())
				})

				It("INTEGRATION should handle a SUBNET message with a VALID CREATE ACTION with INVALID POOL ID", func() {
					testSubnet.Pool = "INVALID_ID"

					subnetMsg, err := json.Marshal(testSubnet)
					if err != nil {
						log.Printf("[ERROR] %s\n", err)
					}
					err = srv.MQ.Send(testExchange, testExchangeType, testRoutingKey, string(subnetMsg), "", replyTo)
					Expect(err).ToNot(HaveOccurred())

					d := <-deliveries
					d.Ack(false)
					err = srv.ProcessMessage(&d)
					Expect(err).ToNot(HaveOccurred())
					r := <-replies
					Expect(string(r.Body)).To(ContainSubstring("[ERROR] getting response from creating subnet"))
				})

				It("INTEGRATION should handle a SUBNET message with a VALID CREATE ACTION with INVALID IP RANGE", func() {
					testSubnet.Start = "123456789"
					testSubnet.End = "987654321"

					subnetMsg, err := json.Marshal(testSubnet)
					Expect(err).ToNot(HaveOccurred())

					err = srv.MQ.Send(testExchange, testExchangeType, testRoutingKey, string(subnetMsg), "", replyTo)
					Expect(err).ToNot(HaveOccurred())

					d := <-deliveries
					d.Ack(false)
					err = srv.ProcessMessage(&d)
					Expect(err).ToNot(HaveOccurred())

					r := <-replies
					Expect(string(r.Body)).To(ContainSubstring("[ERROR] getting response from creating subnet"))

				})
			})

			Context("When a VALID reservation type is requested", func() {

				var testReservation models.IPAMReservationMsg
				var pool resources.PoolV1
				var subnet resources.SubnetV1

				BeforeEach(func() {
					testReservation.Name = "VALID-RESERVATION"
					testReservation.ObjectType = models.ReservationType
					testReservation.Action = models.CreateAction

					pool = resources.PoolV1{Name: "POOL-FOR-RESERVATION-TESTS"}
					pool, err = srv.IPAM.CreateShowPool(pool)
					Expect(err).ToNot(HaveOccurred())

					subnet = resources.SubnetV1{
						Name:  "SUBNET-FOR-RESERVATION-TESTS",
						Pool:  pool.ID,
						Start: "192.168.1.10",
						End:   "192.168.1.10",
					}

					subnet, err = srv.IPAM.CreateShowSubnet(pool.ID, subnet)
					Expect(err).ToNot(HaveOccurred())
					testReservation.Subnet = subnet.ID
				})

				AfterEach(func() {

					_, err = srv.IPAM.DeletePool(pool.ID, pool)
					Expect(err).ToNot(HaveOccurred())

				})

				It("INTEGRATION should handle a RESERVATION message with a VALID CREATE ACTION", func() {
					reservationMsg, err := json.Marshal(testReservation)
					Expect(err).ToNot(HaveOccurred())

					err = srv.MQ.Send(testExchange, testExchangeType, testRoutingKey, string(reservationMsg), "", replyTo)
					Expect(err).ToNot(HaveOccurred())

					d := <-deliveries
					d.Ack(false)
					err = srv.ProcessMessage(&d)
					Expect(err).ToNot(HaveOccurred())
					r := <-replies
					Expect(string(r.Body)).ToNot(ContainSubstring("[ERROR]"))
				})

				It("INTEGRATION should handle a RESERVATION message with a VALID CREATE ACTION with INVALID SUBNET ID", func() {
					testReservation.Subnet = "INVALID_ID"
					reservationMsg, err := json.Marshal(testReservation)
					Expect(err).ToNot(HaveOccurred())

					err = srv.MQ.Send(testExchange, testExchangeType, testRoutingKey, string(reservationMsg), "", replyTo)
					Expect(err).ToNot(HaveOccurred())

					d := <-deliveries
					d.Ack(false)

					err = srv.ProcessMessage(&d)
					Expect(err).ToNot(HaveOccurred())

					r := <-replies
					Expect(string(r.Body)).To(ContainSubstring("[ERROR] getting response from creating reservation"))
				})

				It("INTEGRATION should handle a RESERVATION message with an INVALID ACTION", func() {
					testReservation.Action = "BAD_ACTION"
					reservationMsg, err := json.Marshal(testReservation)
					Expect(err).ToNot(HaveOccurred())

					err = srv.MQ.Send(testExchange, testExchangeType, testRoutingKey, string(reservationMsg), "", replyTo)
					Expect(err).ToNot(HaveOccurred())

					d := <-deliveries
					d.Ack(false)

					err = srv.ProcessMessage(&d)
					Expect(err).To(HaveOccurred())
				})

				It("INTEGRATION should handle a request if Subnet is full", func() {
					reservationMsg, err := json.Marshal(testReservation)
					Expect(err).ToNot(HaveOccurred())

					err = srv.MQ.Send(testExchange, testExchangeType, testRoutingKey, string(reservationMsg), "", replyTo)
					Expect(err).ToNot(HaveOccurred())

					d := <-deliveries
					d.Ack(false)
					err = srv.ProcessMessage(&d)
					Expect(err).ToNot(HaveOccurred())
					r := <-replies
					Expect(string(r.Body)).ToNot(ContainSubstring("[ERROR]"))

					// Create a second reservation, expect it to fail
					testReservation.Name = "very coooooooooooooool"
					reservationMsg, err = json.Marshal(testReservation)
					Expect(err).ToNot(HaveOccurred())

					err = srv.MQ.Send(testExchange, testExchangeType, testRoutingKey, string(reservationMsg), "", replyTo)
					Expect(err).ToNot(HaveOccurred())

					d = <-deliveries
					d.Ack(false)
					err = srv.ProcessMessage(&d)
					Expect(err).ToNot(HaveOccurred())
					r = <-replies
					Expect(string(r.Body)).To(ContainSubstring("[ERROR] getting response from creating reservation"))
				})

			})

			Context("When an INVALID type is requested", func() {
				It("INTEGRATION should returns error", func() {
					testMessage = `{"type":"BAD_TYPE","action":"create","name": "TEST"}`

					err = srv.MQ.Send(testExchange, testExchangeType, testRoutingKey, testMessage, "", replyTo)
					Expect(err).ToNot(HaveOccurred())

					d := <-deliveries
					d.Ack(false)

					err = srv.ProcessMessage(&d)
					Expect(err).ToNot(HaveOccurred())

					r := <-replies
					Expect(string(r.Body)).To(ContainSubstring("[ERROR] Unknown ObjectType"))

				})
			})

		})
	})
})
