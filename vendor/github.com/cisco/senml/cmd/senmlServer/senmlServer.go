package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"github.com/cisco/senml"
	"hash/crc32"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

// TODO
// listen on UDP , TCP
// write to multiple locations
// write to COAP post
// listen on COAP
// test http with something like:  curl --data @data.json http://localhost:8000/data
// curl --data-binary @dagta.cbor http://localhost:8001/data

var doVerbosePtr = flag.Bool("v", false, "verbose")

var doIndentPtr = flag.Bool("i", false, "indent output")
var doPrintPtr = flag.Bool("print", false, "print output to stdout")
var doExpandPtr = flag.Bool("expand", false, "expand SenML records")

var httpPort = flag.Int("http", 0, "port to list for http on")
var postUrl = flag.String("post", "", "URL to HTTP POST output to")
var kafkaUrl = flag.String("kafka", "", "URL to for Apache Kafka Broker to send data to")

var topic = flag.String("topic", "senml", "Apache Kafka topic or InfluxDB series name ")

var doJsonPtr = flag.Bool("json", false, "output JSON formatted SenML ")
var doCborPtr = flag.Bool("cbor", false, "output CBOR formatted SenML ")
var doXmlPtr = flag.Bool("xml", false, "output XML formatted SenML ")
var doCsvPtr = flag.Bool("csv", false, "output CSV formatted SenML ")
var doMpackPtr = flag.Bool("mpack", false, "output MessagePack formatted SenML ")
var doLinpPtr = flag.Bool("linp", false, "output InfluxDB LineProtcol formatted SenML ")

var doIJsonStreamPtr = flag.Bool("ijsons", false, "input JSON formatted SenML stream")
var doIJsonLinePtr = flag.Bool("ijsonl", false, "input JSON formatted SenML lines")
var doIXmlPtr = flag.Bool("ixml", false, "input XML formatted SenML ")
var doICborPtr = flag.Bool("icbor", false, "input CBOR formatted SenML ")
var doIMpackPtr = flag.Bool("impack", false, "input MessagePack formatted SenML ")

var kafkaConn net.Conn = nil
var kafkaReqNumber uint32 = 1

func decodeTimed(msg []byte) (senml.SenML, error) {
	var s senml.SenML
	var err error

	var format senml.Format = senml.JSON
	switch {
	case *doIJsonStreamPtr:
		format = senml.JSON
	case *doIJsonLinePtr:
		format = senml.JSON
	case *doICborPtr:
		format = senml.CBOR
	case *doIXmlPtr:
		format = senml.XML
	case *doIMpackPtr:
		format = senml.MPACK
	}

	s, err = senml.Decode(msg, format) // TODO

	return s, err
}

func outputData(data []byte) error {
	// print the output

	if *doPrintPtr {
		fmt.Print(string(data))
	}

	if kafkaConn != nil {
		//fmt.Println( "sending to Kafka=<" + string(data) + ">" )

		kTopic := []byte(string(*topic))
		clientName := []byte("SenMLCat-0.1")
		partition := 0 // TODO - does this need to be settable from CLI
		reqID := kafkaReqNumber
		kafkaReqNumber += 1

		clientNameLen := len(clientName)
		topicLen := len(kTopic)
		dataLen := len(data)

		var totalLen int = 8 + 56 + clientNameLen + topicLen + dataLen // TODO
		msg := make([]byte, totalLen)

		// protcol doc at https://cwiki.apache.org/confluence/display/KAFKA/A+Guide+To+The+Kafka+Protocol
		l := 0
		// generic size header for req or response
		totalLenLoc := l
		l += 4 // request size after theses bytes

		// generic request header
		binary.BigEndian.PutUint16(msg[l:], uint16(0))
		l += 2 // ApiKey - int16 (ProduceRequest = 0)
		binary.BigEndian.PutUint16(msg[l:], uint16(1))
		l += 2 // ApiVersion  - int16 (curent = 0 but code uses 1 TODO)
		binary.BigEndian.PutUint32(msg[l:], uint32(reqID))
		l += 4 // CorrelationID - int32
		binary.BigEndian.PutUint16(msg[l:], uint16(clientNameLen))
		l += 2 // ClientID Length - int16
		copy(msg[l:], clientName)
		l += clientNameLen // ClientID

		// produce request header
		binary.BigEndian.PutUint16(msg[l:], uint16(1))
		l += 2 // ReqAcks - int16 TODO
		binary.BigEndian.PutUint32(msg[l:], uint32(1500))
		l += 4 // Timeout - int32 - in milli seconds
		binary.BigEndian.PutUint32(msg[l:], uint32(1))
		l += 4 // Array Lenght (for each TopicName)
		binary.BigEndian.PutUint16(msg[l:], uint16(topicLen))
		l += 2 // topic length - int16
		copy(msg[l:], kTopic)
		l += topicLen // topic
		binary.BigEndian.PutUint32(msg[l:], uint32(1))
		l += 4 // Array Length (for each Partition)
		binary.BigEndian.PutUint32(msg[l:], uint32(partition))
		l += 4 // Partion - int32
		msgSetLenLoc := l
		l += 4 // msg set size - int32

		// message Set Header
		binary.BigEndian.PutUint64(msg[l:], uint64(0))
		l += 8 // offset - int64 - any value in produce
		msgLenLoc := l
		l += 4 // message size - int32

		// message
		checksumLoc := l
		l += 4 // CRC - intt32 - compute over rest of message
		msg[l] = uint8(0)
		l += 1 // magic - int8 - current = 0
		msg[l] = uint8(0)
		l += 1 // attributes - int8 - value =0

		// key - bytes - can be null
		binary.BigEndian.PutUint32(msg[l:], uint32(0xFFFFFFFF))
		l += 4 // key length - int32 - (-1 for NULL)

		// value - bytes
		binary.BigEndian.PutUint32(msg[l:], uint32(dataLen))
		l += 4 // data length - int32
		copy(msg[l:], data)
		l += dataLen // message data

		// backpatch in lengths
		binary.BigEndian.PutUint32(msg[msgLenLoc:], uint32(l-msgLenLoc-4))
		binary.BigEndian.PutUint32(msg[msgSetLenLoc:], uint32(l-msgSetLenLoc-4))
		binary.BigEndian.PutUint32(msg[totalLenLoc:], uint32(l-totalLenLoc-4))

		// backpatch in checksum
		checksum := crc32.ChecksumIEEE(msg[checksumLoc+4 : l])
		binary.BigEndian.PutUint32(msg[checksumLoc:], uint32(checksum))

		if l != totalLen {
			fmt.Println("Header Len ", l-(clientNameLen+topicLen+dataLen))
			panic("assumed kafka mesg header length wrong ")
		}

		n, err := kafkaConn.Write(msg)
		if err != nil {
			fmt.Println("Write to kafka ", string(*kafkaUrl), " got error", err)
			return err
		}
		if n != totalLen {
			return errors.New("Coud not write request to Kafka server") // TODO
		}

		// read a length
		lenBuf := make([]byte, 4)
		n, err = kafkaConn.Read(lenBuf)
		if err != nil {
			return errors.New("Coud not read response from Kafka server")
		}
		if n != 4 {
			return errors.New("Coud not read response from Kafka server")
		}

		respLen := binary.BigEndian.Uint32(lenBuf)
		buf := make([]byte, respLen)

		n, err = kafkaConn.Read(buf)
		if err != nil {
			return errors.New("Coud not read response from Kafka server")
		}
		if n != int(respLen) {
			return errors.New("Coud not read response from Kafka server")
		}

		l = 0
		respReqID := binary.BigEndian.Uint32(buf[l:])
		l += 4 // CorrelationID int32
		respNumTopics := binary.BigEndian.Uint32(buf[l:])
		l += 4 // Aray Length - int32 (for each topic)
		for t := uint32(0); t < respNumTopics; t += 1 {
			topicLen := int(binary.BigEndian.Uint16(buf[l:]))
			l += 2 // TopicName length - int16
			topic := string(buf[l : l+topicLen])
			l += topicLen // Topic string
			respNumPartitions := binary.BigEndian.Uint32(buf[l:])
			l += 4 // Array length - int32 (for each partition)
			for p := uint32(0); p < respNumPartitions; p += 1 {
				respPartition := binary.BigEndian.Uint32(buf[l:])
				l += 4 // Partition - int32
				respError := binary.BigEndian.Uint16(buf[l:])
				l += 2 // Error code - int16 ( 0 is no error)
				respOffset := binary.BigEndian.Uint64(buf[l:])
				l += 8 // offset - int64

				if respError != 0 {
					fmt.Println("Kafka response err=", respError,
						" part=", respPartition, " topic=", topic, " respReqID=", respReqID, " respOffset=", respOffset)

					switch respError {
					case 0: // no error
					case 0xFFFF:
						return errors.New("Kafka unknown error") // unknown error
					case 1:
						return errors.New("Kafka Error OffsetOutOfRange")
					case 2:
						return errors.New("Kafka Error InvalidMessage")
					case 3:
						return errors.New("Kafka Error UnknownTopicOrPartition")
					case 4:
						return errors.New("Kafka Error InvalidMessageSize")
					case 5:
						return errors.New("Kafka Error LeaderNotAvailable")
					case 6:
						return errors.New("Kafka Error NotLeaderForPartition")
					case 7:
						return errors.New("Kafka Error RequestTimedOut")
					case 8:
						return errors.New("Kafka Error BrokerNotAvailable")
					case 9:
						return errors.New("Kafka Error ReplicaNotAvailable")
					case 10:
						return errors.New("Kafka Error MessageSizeTooLarge")
					case 11:
						return errors.New("Kafka Error StaleControllerEpochCode")
					case 12:
						return errors.New("Kafka Error OffsetMetadataTooLargeCode")
					case 14:
						return errors.New("Kafka Error GroupLoadInProgressCode")
					case 15:
						return errors.New("Kafka Error GroupCoordinatorNotAvailableCode")
					case 16:
						return errors.New("Kafka Error NotCoordinatorForGroupCode")
					case 17:
						return errors.New("Kafka Error InvalidTopicCode")
					case 18:
						return errors.New("Kafka Error RecordListTooLargeCode")
					case 19:
						return errors.New("Kafka Error NotEnoughReplicasCode")
					case 20:
						return errors.New("Kafka Error NotEnoughReplicasAfterAppendCode")
					case 21:
						return errors.New("Kafka Error InvalidRequiredAcksCode")
					case 22:
						return errors.New("Kafka Error IllegalGenerationCode")
					case 23:
						return errors.New("Kafka Error InconsistentGroupProtocolCode")
					case 24:
						return errors.New("Kafka Error InvalidGroupIdCode")
					case 25:
						return errors.New("Kafka Error UnknownMemberIdCode")
					case 26:
						return errors.New("Kafka Error InvalidSessionTimeoutCode")
					case 27:
						return errors.New("Kafka Error RebalanceInProgressCode")
					case 28:
						return errors.New("Kafka Error InvalidCommitOffsetSizeCode")
					case 29:
						return errors.New("Kafka Error TopicAuthorizationFailedCode")
					case 30:
						return errors.New("Kafka Error GroupAuthorizationFailedCode")
					case 31:
						return errors.New("Kafka Error ClusterAuthorizationFailedCode")
					}
				}

			}
		}
	}

	if len(*postUrl) != 0 {
		fmt.Println("PostURL=<" + string(*postUrl) + ">")
		buffer := bytes.NewBuffer(data)
		_, err := http.Post(string(*postUrl), "application/senml+json", buffer)
		if err != nil {
			fmt.Println("Post to", string(*postUrl), " got error", err.Error())
			return err
		}
	}

	return nil
}

func processData(dataIn []byte) error {
	var s senml.SenML
	var err error

	//fmt.Println( "DataIn:", dataIn )

	s, err = decodeTimed(dataIn)
	if err != nil {
		fmt.Println("Decode of SenML failed")
		return err
	}

	//fmt.Println( "Senml:", senml.Records )
	if *doExpandPtr {
		s = senml.Normalize(s)
	}

	var dataOut []byte

	options := senml.OutputOptions{}
	if *doIndentPtr {
		options.PrettyPrint = *doIndentPtr
	}
	options.Topic = string(*topic)

	var format senml.Format = senml.JSON
	switch {
	case *doJsonPtr:
		format = senml.JSON
	case *doCborPtr:
		format = senml.CBOR
	case *doXmlPtr:
		format = senml.XML
	case *doCsvPtr:
		format = senml.CSV
	case *doMpackPtr:
		format = senml.MPACK
	case *doLinpPtr:
		format = senml.LINEP
	}

	dataOut, err = senml.Encode(s, format, options)
	if err != nil {
		fmt.Println("Encode of SenML failed")
		return err
	}

	err = outputData(dataOut)
	if err != nil {
		fmt.Println("Output of SenML failed:", err)
		return err
	}

	return nil
}

func httpReqHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println( "HTTP request to ",  r.URL.Path )
	//fmt.Println( "Method: ",  r.Method )

	// defer r.Body.Close() // not needed

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic("Problem reading HTTP body")
	}

	if *doVerbosePtr {
		fmt.Println("HTTP Body: ", body)
	}

	err = processData(body)
	if err != nil {
		http.Error(w, err.Error(), 400)
	}
}

func main() {
	var err error

	flag.Parse()

	if len(*kafkaUrl) != 0 {
		kafkaConn, err = net.DialTimeout("tcp", *kafkaUrl, 2500*time.Millisecond)
		if err != nil {
			fmt.Println("error connecting to kafka broker", err)
			os.Exit(1)
		}
		defer kafkaConn.Close()
	}

	http.HandleFunc("/", httpReqHandler)
	http.ListenAndServe(":"+strconv.Itoa(*httpPort), nil)
}
