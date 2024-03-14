package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"go.bug.st/serial"
	"log"
	"math"
	"net/http"
	"os"
)

type Packet struct {
	data        []byte
	fromId      uint16
	toId        uint16
	dataSize    uint16
	messageId   uint8
	messageType uint8
}

type Body struct {
	Sensors  []Sensor `json:"sensors"`
	DeviceId uint64   `json:"device_id"`
}

type Sensor struct {
	SensorId uint64  `json:"sensor_id"`
	Value    float64 `json:"value"`
}

func main() {

	arg := os.Args[1]
	serverPort := 5000
	userId := 1
	requestUrl := fmt.Sprintf("http://localhost:%d/new/user/%d/measurements", serverPort, userId)

	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}

	if len(ports) == 0 {
		log.Fatal("No ports found")
	}

	portFound := false
	for _, port := range ports {
		if port == arg {
			portFound = true
			break
		}
	}

	if !portFound {
		log.Fatal("Requested port is not available")
		return
	}

	mode := &serial.Mode{BaudRate: 9600}
	port, err := serial.Open(arg, mode)
	if err != nil {
		log.Fatal(err)
		return
	}

	buff := make([]byte, 256)
	for {
		n, err := port.Read(buff)
		if err != nil {
			log.Fatal(err)
			break
		}
		if n == 0 {
			continue
		}

		packet, err := makePacket(buff)
		if err != nil {
			log.Fatal(err)
			continue
		}

		body, err := makeBody(packet)
		if err != nil {
			log.Fatal(err)
			continue
		}

		_, err = sendRequest(requestUrl, body)
		if err != nil {
			log.Fatal(err)
			continue
		}
	}
}

func makePacket(buff []byte) (Packet, error) {
	if len(buff) < 8 {
		return Packet{}, errors.New("read packet is not big enough for header")
	}

	fromId := binary.LittleEndian.Uint16([]byte{buff[0], buff[1]}) - 2
	toId := binary.LittleEndian.Uint16([]byte{buff[2], buff[4]})
	messageId := buff[5]
	messageType := buff[6]
	dataSize := binary.LittleEndian.Uint16([]byte{buff[7], buff[8]})
	data := buff[9:dataSize]

	return Packet{
		fromId:      fromId,
		toId:        toId,
		messageId:   messageId,
		messageType: messageType,
		dataSize:    dataSize,
		data:        data,
	}, nil
}

func makeBody(packet Packet) (Body, error) {
	if packet.messageType != 0 {
		return Body{}, errors.New("unsupported message type")
	}

	bits := binary.LittleEndian.Uint64(packet.data)
	float := math.Float64frombits(bits)

	return Body{
		DeviceId: uint64(packet.fromId),
		Sensors: []Sensor{
			{
				SensorId: 0,
				Value:    float,
			},
		},
	}, nil
}

func sendRequest(url string, body Body) (*http.Response, error) {
	val, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(val))
	return resp, err

}
