package server_test

import (
	"encoding/json"
	"os"

	"github.com/RackHD/ipam/resources"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/RackHD/voyager-ipam-service/server"
	"github.com/RackHD/voyager-utilities/models"
	"github.com/RackHD/voyager-utilities/random"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("INTEGRATION", func() {
	Describe("AMQP Message Handling", func() {
		var srv *server.Server
		var amqpServer string
		var err error

		BeforeEach(func() {
			amqpServer = os.Getenv("RABBITMQ_URL")
			srv = server.NewServer(amqpServer, "127.0.0.1:8000")
			Expect(srv.MQ).ToNot(BeNil())
		})

		AfterEach(func() {
			srv.MQ.Close()
		})

		Context("when a ip request is recevied", func() {
			var pool resources.PoolV1
			var subnet resources.SubnetV1

			BeforeEach(func() {
				pool = resources.PoolV1{Name: "POOL-FOR-LEASE-TESTS"}
				pool, err = srv.IPAM.CreateShowPool(pool)
				Expect(err).ToNot(HaveOccurred())

				subnet = resources.SubnetV1{
					Name:  "SUBNET-FOR-LEASE-TESTS",
					Pool:  pool.ID,
					Start: "192.168.1.10",
					End:   "192.168.1.11",
				}

				subnet, err = srv.IPAM.CreateShowSubnet(pool.ID, subnet)
				Expect(err).ToNot(HaveOccurred())

				err = srv.Start()
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				_, err = srv.IPAM.DeletePool(pool.ID, pool)
				Expect(err).ToNot(HaveOccurred())

				err = srv.Stop()
				Expect(err).ToNot(HaveOccurred())
			})

			It("INTEGRATION should return a valid ip", func() {
				ipamQueueName := random.RandQueue()
				_, sendDeliveries, err := srv.MQ.Listen(models.IpamExchange, models.IpamExchangeType, ipamQueueName, models.IpamSendQueue, "")
				Expect(err).ToNot(HaveOccurred())

				ipReq := models.IpamLeaseReq{
					Action:   models.RequestIPAction,
					SubnetID: subnet.ID,
				}
				ipReqBytes, err := json.Marshal(ipReq)
				Expect(err).ToNot(HaveOccurred())
				err = srv.MQ.Send(models.IpamExchange, models.IpamExchangeType, models.IpamReceiveQueue, string(ipReqBytes), "", models.IpamSendQueue)
				Expect(err).ToNot(HaveOccurred())

				message := <-sendDeliveries
				message.Ack(false)
				var resp models.IpamLeaseResp
				err = json.Unmarshal(message.Body, &resp)
				log.Infof("resp is %+v", resp)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Failed).To(BeFalse())
				Expect(resp.Error).To(BeEmpty())
				Expect(resp.IP).To(Equal("192.168.1.10"))
				Expect(resp.Reservation).ToNot(BeEmpty())
			})

		})
	})
})
