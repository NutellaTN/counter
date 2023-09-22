package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"time"
	"log"
	device "github.com/d2r2/go-hd44780"
	i2c "github.com/d2r2/go-i2c"







	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
)

var cli mqtt.Client

const (
	mqttUrl = "tcp://127.0.0.1:1883"
	topic   = "$hw/events/device/counter/twin/update"
)

const (
        ON = iota
        OFF
)

type Counter struct {
        status chan int
        handle func (int)
}



//BaseMessage the base struct of event message
type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

//TwinValue the struct of twin value
type TwinValue struct {
	Value    *string        `json:"value, omitempty"`
	Metadata *ValueMetadata `json:"metadata,omitempty"`
}

//ValueMetadata the meta of value
type ValueMetadata struct {
	Timestamp int64 `json:"timestamp, omitempty"`
}

//TypeMetadata the meta of value type
type TypeMetadata struct {
	Type string `json:"type,omitempty"`
}

//TwinVersion twin version
type TwinVersion struct {
	CloudVersion int64 `json:"cloud"`
	EdgeVersion  int64 `json:"edge"`
}

//MsgTwin the struct of device twin
type MsgTwin struct {
	Expected        *TwinValue    `json:"expected,omitempty"`
	Actual          *TwinValue    `json:"actual,omitempty"`
	Optional        *bool         `json:"optional,omitempty"`
	Metadata        *TypeMetadata `json:"metadata,omitempty"`
	ExpectedVersion *TwinVersion  `json:"expected_version,omitempty"`
	ActualVersion   *TwinVersion  `json:"actual_version,omitempty"`
}

//DeviceTwinUpdate the struct of device twin update
type DeviceTwinUpdate struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

//createActualUpdateMessage function is used to create the device twin update message
func createActualUpdateMessage(actualValue string) DeviceTwinUpdate {
	var deviceTwinUpdateMessage DeviceTwinUpdate
	actualMap := map[string]*MsgTwin{"status": {Actual: &TwinValue{Value: &actualValue}, Metadata: &TypeMetadata{Type: "Updated"}}}
	deviceTwinUpdateMessage.Twin = actualMap
	return deviceTwinUpdateMessage
}

func publishToMqtt(data int) {
	updateMessage := createActualUpdateMessage(strconv.Itoa(data))
	twinUpdateBody, _ := json.Marshal(updateMessage)

	token := cli.Publish(topic, 0, false, twinUpdateBody)

	if token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}
}

func connectToMqtt() mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttUrl)

	cli = mqtt.NewClient(opts)

	token := cli.Connect()
	if token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}

	return cli
}





func (counter *Counter) runDevice(interrupt chan struct{}) {
	data := 0
	i2c, err := i2c.NewI2C(0x27, 1)
	if err != nil { log.Fatal(err) }
	defer i2c.Close()
	lcd, err := device.NewLcd(i2c, device.LCD_16x2)
	if err != nil { log.Fatal(err) }
	lcd.BacklightOn()
	lcd.Clear()
	for {
		select {
		case <-interrupt:
			err = lcd.BacklightOff()
  			if err != nil { log.Fatal(err) }
			counter.handle(0)
			return
		default:
			data++
			err = lcd.ShowMessage(strconv.Itoa(data), device.SHOW_LINE_1)
			if err != nil { log.Fatal(err) }
			counter.handle(data)
			fmt.Println("Counter value:", data)
			time.Sleep(1 * time.Second)
		}
	}
}

func (counter *Counter) initDevice() {
	interrupt := make(chan struct{})

	for {
		select {
		case status := <-counter.status:
			if status == ON {
				go counter.runDevice(interrupt)
			}
			if status == OFF {
				interrupt <- struct{}{}
			}
		}
	}
}

func (counter *Counter) TurnOn() {
	counter.status <- ON
}

func (counter *Counter) TurnOff() {
	counter.status <- OFF
}

func NewCounter(h func (x int)) *Counter {
	counter := &Counter{
		status: make(chan int),
		handle: h,
	}

	go counter.initDevice()

	return counter
}

func CloseCounter(counter *Counter) {
	close(counter.status)
}



func main() {
	stopchan := make(chan os.Signal)
	signal.Notify(stopchan, syscall.SIGINT, syscall.SIGKILL)
	defer close(stopchan)

	cli = connectToMqtt()

	// Link to pseudo device counter
	//ctr := counter.NewCounter(publishToMqtt)
	ctr := NewCounter(publishToMqtt)
	current_status := "OFF"

	token := cli.Subscribe(topic+"/document", 0, func(client mqtt.Client, msg mqtt.Message) {
		Update := &types.DeviceTwinDocument{}
		err := json.Unmarshal(msg.Payload(), Update)
		if err != nil {
			fmt.Printf("Unmarshal error: %v\n", err)
		}

		cmd := *Update.Twin["status"].CurrentState.Expected.Value

		if cmd == "ON" && cmd != current_status {
			ctr.TurnOn()
			fmt.Printf("turn on counter.\n")
		}

		if cmd == "OFF" && cmd != current_status {
			ctr.TurnOff()
			fmt.Printf("turn off counter.\n")
		}

		current_status = cmd
	})

	if token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}

	select {
	case <-stopchan:
		fmt.Printf("Interrupt, exit.\n")
		break
	}
}
