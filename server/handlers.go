package server

import (
  "encoding/json"
  "fmt"
  "log"

  "github.com/RackHD/ipam/resources"
  "github.com/RackHD/voyager-utilities/models"
  "github.com/streadway/amqp"
)

// IPHandler handles reservation rabbitmq request
func (s *Server) IPHandler(subnetID string) (string, error) {

  reservation, err := s.IPAM.CreateShowReservation(subnetID, resources.ReservationV1{})
  if err != nil {
    resp := fmt.Errorf("[ERROR] getting response from creating reservation %s", err)
    log.Println(resp)
    return resp.Error(), resp
  }

  leases, err := s.IPAM.IndexLeases(reservation.ID)
  if err != nil || len(leases.Leases) == 0 {
    resp := fmt.Errorf("[ERROR] getting lease, %s", err)
    log.Println(resp)
    return resp.Error(), resp
  }

  resp := models.IpamLeaseResp{
    IP:          leases.Leases[0].Address,
    Reservation: reservation.ID,
  }
  response, err := json.Marshal(resp)
  if err != nil {
    resp := fmt.Errorf("[ERROR] marshalling create reservation response %s", err)
    log.Println(resp)
    return resp.Error(), resp
  }
  return string(response), nil
}

// PoolHandler handles pool rabbitmq request
func (s *Server) PoolHandler(d *amqp.Delivery) (string, error) {
  msg := models.IPAMPoolMsg{}
  if err := json.Unmarshal(d.Body, &msg); err != nil {
    log.Println(err)
    return fmt.Sprintf("[ERROR] Unmarshaling AMQP message body %s", err), err
  }

  switch msg.Action {
  case models.CreateAction:
    pool := resources.PoolV1{
      ID:       msg.ID,
      Name:     msg.Name,
      Tags:     msg.Tags,
      Metadata: msg.Metadata,
    }
    pool, err := s.IPAM.CreateShowPool(pool)
    if err != nil {
      resp := fmt.Errorf("[ERROR] getting response from creating pool %s", err)
      log.Println(resp)
      return resp.Error(), resp
    }

    var response []byte
    response, err = json.Marshal(pool)
    if err != nil {
      resp := fmt.Errorf("[ERROR] marshalling create pool response %s", err)
      log.Println(resp)
      return resp.Error(), resp
    }
    return string(response), nil

  default:
    resp := fmt.Errorf("[ERROR] Unknown Action: %s", msg.Action)
    log.Println(resp)
    return resp.Error(), resp

  }
}

// SubnetHandler handles subnet rabbitmq request
func (s *Server) SubnetHandler(d *amqp.Delivery) (string, error) {
  msg := models.IPAMSubnetMsg{}
  if err := json.Unmarshal(d.Body, &msg); err != nil {
    log.Println(err)
    return fmt.Sprintf("[ERROR] Unmarshaling AMQP message body %s", err), err
  }

  switch msg.Action {
  case models.CreateAction:
    //log.Printf("AMQP-IPAM: %+v", msg)
    subnet := resources.SubnetV1{
      ID:       msg.ID,
      Name:     msg.Name,
      Tags:     msg.Tags,
      Metadata: msg.Metadata,
      Pool:     msg.Pool,
      Start:    msg.Start,
      End:      msg.End,
    }

    subnet, err := s.IPAM.CreateShowSubnet(subnet.Pool, subnet)
    if err != nil {
      resp := fmt.Errorf("[ERROR] getting response from creating subnet %s", err)
      log.Println(resp)
      return resp.Error(), resp
    }

    var response []byte
    response, err = json.Marshal(subnet)
    if err != nil {
      resp := fmt.Errorf("[ERROR] marshalling create subnet response %s", err)
      log.Println(resp)
      return resp.Error(), resp
    }
    return string(response), nil

  default:
    resp := fmt.Errorf("[ERROR] Unknown Action: %s", msg.Action)
    log.Println(resp)
    return resp.Error(), resp
  }
}

// ReservationHandler handles reservation rabbitmq request
func (s *Server) ReservationHandler(d *amqp.Delivery) (string, error) {
  msg := models.IPAMReservationMsg{}
  if err := json.Unmarshal(d.Body, &msg); err != nil {
    log.Println(err)
    return fmt.Sprintf("[ERROR] Unmarshaling AMQP message body %s", err), err
  }

  switch msg.Action {
  case models.CreateAction:
    reservation := resources.ReservationV1{
      ID:       msg.ID,
      Name:     msg.Name,
      Tags:     msg.Tags,
      Metadata: msg.Metadata,
      Subnet:   msg.Subnet,
    }

    reservation, err := s.IPAM.CreateShowReservation(reservation.Subnet, reservation)
    if err != nil {
      resp := fmt.Errorf("[ERROR] getting response from creating reservation %s", err)
      log.Println(resp)
      return resp.Error(), resp
    }

    leases, err := s.IPAM.IndexLeases(reservation.ID)
    if err != nil || len(leases.Leases) != 1 {
      resp := fmt.Errorf("[ERROR] getting lease, %s", err)
      log.Println(resp)
      return resp.Error(), resp
    }

    var response []byte
    response, err = json.Marshal(reservation)
    if err != nil {
      resp := fmt.Errorf("[ERROR] marshalling create reservation response %s", err)
      log.Println(resp)
      return resp.Error(), resp
    }
    return string(response), nil

  case models.DeleteAction:

    response, err := s.IPAM.DeleteReservation(msg.ID, resources.ReservationV1{})
    if err != nil {
      resp := fmt.Errorf("[ERROR] getting response from creating reservation %s", err)
      log.Println(resp)
      return resp.Error(), resp
    }
    return response, nil

  default:
    resp := fmt.Errorf("[ERROR] Unknown Action: %s", msg.Action)
    log.Println(resp)
    return resp.Error(), resp
  }
}

// LeaseHandler handles lease rabbitmq request
func (s *Server) LeaseHandler(d *amqp.Delivery) (string, error) {
  msg := models.IPAMLeaseMsg{}
  if err := json.Unmarshal(d.Body, &msg); err != nil {
    log.Println(err)
    return fmt.Sprintf("[ERROR] Unmarshaling AMQP message body %s", err), err
  }

  switch msg.Action {
  case models.ShowAction:
    leases, err := s.IPAM.IndexLeases(msg.Reservation)
    if err != nil || len(leases.Leases) != 1 {
      resp := fmt.Errorf("[ERROR] getting lease, %s", err)
      log.Println(resp)
      return resp.Error(), resp
    }

    var response []byte
    response, err = json.Marshal(leases.Leases[0])
    if err != nil {
      resp := fmt.Errorf("[ERROR] marshalling lease response %s", err)
      log.Println(resp)
      return resp.Error(), resp
    }
    return string(response), nil

  default:
    resp := fmt.Errorf("[ERROR] Unknown Action: %s", msg.Action)
    log.Println(resp)
    return resp.Error(), resp
  }
}
