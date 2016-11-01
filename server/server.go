package server

import (
	"encoding/json"
	"fmt"

	ipamapi "github.com/RackHD/ipam/client"
	"github.com/RackHD/voyager-utilities/amqp"
	"github.com/RackHD/voyager-utilities/models"
	"github.com/RackHD/voyager-utilities/random"
	log "github.com/sirupsen/logrus"
	samqp "github.com/streadway/amqp"
)

// Server defines amqp server
type Server struct {
	MQ   *amqp.Client
	IPAM *ipamapi.Client
}

// NewServer creates new amqp server
func NewServer(amqpServer, ipamAddress string) *Server {
	server := Server{}
	server.MQ = amqp.NewClient(amqpServer)
	if server.MQ == nil {
		log.Fatalf("Could not connect to RabbitMQ server: %s\n", amqpServer)
	}
	server.IPAM = ipamapi.NewClient(ipamAddress)
	return &server
}

// Start starts the server
func (s *Server) Start() error {
	ipamQueueName := random.RandQueue()
	_, deliveries, err := s.MQ.Listen(models.IpamExchange, models.IpamExchangeType, ipamQueueName, models.IpamReceiveQueue, "")
	if err != nil {
		log.Printf("Count not start the server: %s\n", err)
		return err
	}

	go func() {
		for m := range deliveries {
			log.Printf("got %dB delivery on exchange %s: [%v] %s", len(m.Body), m.Exchange, m.DeliveryTag, m.Body)
			m.Ack(true)
			go s.ProcessMessage(&m)
		}
	}()

	return nil
}

// Stop stops the ipam amqp server
func (s *Server) Stop() error {
	err := s.MQ.Close()
	if err != nil {
		log.Printf("Server Not started: %s\n", err)
	}
	return err
}

// ProcessMessage processes a message
func (s *Server) ProcessMessage(msg *samqp.Delivery) error {
	if msg.Exchange != models.IpamExchange {
		log.Printf("Unknown exchange name: %s\n", msg.Exchange)
		return fmt.Errorf("[ERROR] Unknown exchange name: %s\n", msg.Exchange)
	}
	return s.processIpamService(msg)
}

// processIpamService processes a message from the Ipam exchange
func (s *Server) processIpamService(d *samqp.Delivery) error {
	var req models.IpamLeaseReq

	err := json.Unmarshal(d.Body, &req)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("[ERROR] %s\n", err)
	}
	log.Printf("Unmarshalled message: %s\n", req)

	switch req.Action {
	case models.RequestIPAction:
		resp, err := s.IPHandler(req.SubnetID)
		if err != nil {
			errResp := models.AmqpResp{
				Failed: true,
				Error:  err.Error(),
			}
			r, e := json.Marshal(errResp)
			if e != nil {
				log.Println(e)
				return fmt.Errorf("[ERROR] When marshalling message %s", e)
			}
			resp = string(r)
		}

		err = s.MQ.Send(d.Exchange, models.IpamExchangeType, d.ReplyTo, resp, d.CorrelationId, "")
		if err != nil {
			log.Println(err)
			return fmt.Errorf("failed to send resp to %s due to %s", d.ReplyTo, err)
		}

		return nil

	case models.CreateAction:
		err := s.processIpamCreate(d)
		if err != nil {
			log.Println(err)
		}
		return err

	default:
		log.Println("Unknown Message Action")
		return fmt.Errorf("[ERROR] Unknown Action: %s", req.Action)
	}
}

func (s *Server) processIpamCreate(d *samqp.Delivery) error {
	var amqpIpam models.AMQPMsg

	var response string
	var err error

	if err = json.Unmarshal(d.Body, &amqpIpam); err != nil {
		log.Println(err)
		return fmt.Errorf("[ERROR] %s\n", err)
	}

	log.Printf("Received message on voyager-ipam-service Exchange: %s\n", d.Body)
	log.Printf("Unmarshalled message: %s\n", amqpIpam)

	switch amqpIpam.ObjectType {
	case models.PoolType:

		response, err = s.PoolHandler(d)
		if err != nil {
			log.Warn(err)
		}

	case models.SubnetType:

		response, err = s.SubnetHandler(d)
		if err != nil {
			log.Warn(err)
		}

	case models.ReservationType:

		response, err = s.ReservationHandler(d)
		if err != nil {
			log.Warn(err)
		}

	case models.LeaseType:

		response, err = s.LeaseHandler(d)
		if err != nil {
			log.Warn(err)
		}

	default:
		response = fmt.Sprintf("[ERROR] Unknown ObjectType: %s", amqpIpam.ObjectType)
		log.Warn(response)
	}

	err = s.MQ.Send(d.Exchange, "topic", d.ReplyTo, response, d.CorrelationId, "")
	log.Printf("Message Sent : %s\n", response)
	return err
}
