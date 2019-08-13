package cmd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mainflux/mainflux/tools/mqtt-bench/mqtt"
	"github.com/mainflux/mainflux/tools/mqtt-bench/res"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// Execute - main command
func Execute() {
	if err := benchCmd.Execute(); err != nil {
		log.Fatalf(err.Error())
	}
}

// Config - command optiopns from file configuration
type Config struct {
	Broker     string `toml:"broker"`
	QoS        int    `toml:"qos"`
	Size       int    `toml:"size"`
	Count      int    `toml:"count"`
	Pubs       int    `toml:"pubs"`
	Subs       int    `toml:"subs"`
	Format     string `toml:"format"`
	Quiet      bool   `toml:"quiet"`
	Mtls       bool   `toml:"mtls"`
	Retain     bool   `toml:"retain"`
	SkipTLSVer bool   `toml:"skiptlsver"`
	CA         string `toml:"ca"`
	Channels   string `toml:"channels"`
}

// JSONResults are used to export results as a JSON document
type JSONResults struct {
	Runs   []*res.RunResults `json:"runs"`
	Totals *res.TotalResults `json:"totals"`
}

// Connection represents connection
type connection struct {
	ChannelID string `json:"ChannelID"`
	ThingID   string `json:"ThingID"`
	ThingKey  string `json:"ThingKey"`
	MTLSCert  string `json:"MTLSCert"`
	MTLSKey   string `json:"MTLSKey"`
}

// Connections - representing connections from channels file
type Connections struct {
	Connection []connection
}

var (
	broker     string
	qos        int
	size       int
	count      int
	pubs       int
	subs       int
	format     string
	conf       string
	channels   string
	quiet      bool
	retain     bool
	mtls       bool
	skipTLSVer bool
	ca         string
)

var benchCmd = &cobra.Command{
	Use: "mqtt-bench",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
		}

		runBench()
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	benchCmd.PersistentFlags().StringVarP(&broker, "broker", "b", "tcp://localhost:1883", "address for mqtt broker, for secure use tcps and 8883")
	benchCmd.PersistentFlags().IntVarP(&qos, "qos", "q", 0, "QoS for published messages, values 0 1 2")
	benchCmd.PersistentFlags().IntVarP(&size, "size", "s", 100, "Size of message payload bytes")
	benchCmd.PersistentFlags().IntVarP(&count, "count", "n", 100, "Number of messages sent per publisher")
	benchCmd.PersistentFlags().IntVarP(&subs, "subs", "", 10, "Number of subscribers")
	benchCmd.PersistentFlags().IntVarP(&pubs, "pubs", "", 10, "Number of publishers")
	benchCmd.PersistentFlags().StringVarP(&format, "format", "f", "text", "Output format: text|json")
	benchCmd.PersistentFlags().StringVarP(&conf, "config", "g", "config.toml", "config file default is config.toml")
	benchCmd.PersistentFlags().StringVarP(&channels, "channels", "", "channels.toml", "config file for channels")
	benchCmd.PersistentFlags().StringVarP(&ca, "ca", "", "ca.crt", "CA file")
	benchCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "", false, "Supress messages")
	benchCmd.PersistentFlags().BoolVarP(&retain, "retain", "r", false, "Retain mqtt messages")
	benchCmd.PersistentFlags().BoolVarP(&mtls, "mtls", "m", false, "Use mtls for connection")
	benchCmd.PersistentFlags().BoolVarP(&skipTLSVer, "skipTLSVer", "t", false, "Skip tls verification")

}

func initConfig() {

	if conf != "" {
		viper.SetConfigFile(conf)
	}
	c := Config{}
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {

		err = viper.Unmarshal(&c)
		if err != nil {
			log.Printf("failed to load config - %s", err.Error())
		}
		log.Printf("config file: %s", viper.ConfigFileUsed())
	}
}

func runBench() {
	var wg sync.WaitGroup
	var err error

	subTimes := make(res.SubTimes)

	if pubs < 1 && subs < 1 {
		log.Fatal("Invalid arguments")
	}

	var caByte []byte
	if mtls {
		caFile, err := os.Open(ca)
		defer caFile.Close()
		if err != nil {
			fmt.Println(err)
		}

		caByte, _ = ioutil.ReadAll(caFile)
	}

	c := Connections{}

	loadChansConfig(&channels, &c)
	connections := c.Connection

	resCh := make(chan *res.RunResults)
	done := make(chan bool)

	start := time.Now()
	n := len(connections)
	var cert tls.Certificate
	for i := 0; i < subs; i++ {

		con := connections[i%n]

		if mtls {
			cert, err = tls.X509KeyPair([]byte(con.MTLSCert), []byte(con.MTLSKey))
			if err != nil {
				log.Fatal(err)
			}
		}

		c := &mqtt.Client{
			ID:         strconv.Itoa(i),
			BrokerURL:  broker,
			BrokerUser: con.ThingID,
			BrokerPass: con.ThingKey,
			MsgTopic:   getTestTopic(con.ChannelID),
			MsgSize:    size,
			MsgCount:   count,
			MsgQoS:     byte(qos),
			Quiet:      quiet,
			Mtls:       mtls,
			SkipTLSVer: skipTLSVer,
			CA:         caByte,
			ClientCert: cert,
			Retain:     retain,
		}
		wg.Add(1)
		go c.RunSubscriber(&wg, &subTimes, &done, mtls)
	}
	wg.Wait()

	for i := 0; i < pubs; i++ {

		con := connections[i%n]

		if mtls {
			cert, err = tls.X509KeyPair([]byte(con.MTLSCert), []byte(con.MTLSKey))
			if err != nil {
				log.Fatal(err)
			}
		}

		c := &mqtt.Client{
			ID:         strconv.Itoa(i),
			BrokerURL:  broker,
			BrokerUser: con.ThingID,
			BrokerPass: con.ThingKey,
			MsgTopic:   getTestTopic(con.ChannelID),
			MsgSize:    size,
			MsgCount:   count,
			MsgQoS:     byte(qos),
			Quiet:      quiet,
			Mtls:       mtls,
			SkipTLSVer: skipTLSVer,
			CA:         caByte,
			ClientCert: cert,
			Retain:     retain,
		}
		go c.RunPublisher(resCh, mtls)
	}

	// collect the results
	var results []*res.RunResults
	if pubs > 0 {
		results = make([]*res.RunResults, pubs)
	}

	for i := 0; i < pubs; i++ {
		results[i] = <-resCh
	}

	totalTime := time.Now().Sub(start)
	totals := calculateTotalResults(results, totalTime, &subTimes)
	if totals == nil {
		return
	}
	// print stats
	printResults(results, totals, format, quiet)
}
func calculateTotalResults(results []*res.RunResults, totalTime time.Duration, subTimes *res.SubTimes) *res.TotalResults {
	if results == nil || len(results) < 1 {
		return nil
	}
	totals := new(res.TotalResults)
	totals.TotalRunTime = totalTime.Seconds()
	var subTimeRunResults res.RunResults
	msgTimeMeans := make([]float64, len(results))
	msgTimeMeansDelivered := make([]float64, len(results))
	msgsPerSecs := make([]float64, len(results))
	runTimes := make([]float64, len(results))
	bws := make([]float64, len(results))

	totals.MsgTimeMin = results[0].MsgTimeMin
	for i, res := range results {

		if len(*subTimes) > 0 {
			times := mat.NewDense(1, len((*subTimes)[res.ID]), (*subTimes)[res.ID])

			subTimeRunResults.MsgTimeMin = mat.Min(times)
			subTimeRunResults.MsgTimeMax = mat.Max(times)
			subTimeRunResults.MsgTimeMean = stat.Mean((*subTimes)[res.ID], nil)
			subTimeRunResults.MsgTimeStd = stat.StdDev((*subTimes)[res.ID], nil)

		}
		res.MsgDelTimeMin = subTimeRunResults.MsgTimeMin
		res.MsgDelTimeMax = subTimeRunResults.MsgTimeMax
		res.MsgDelTimeMean = subTimeRunResults.MsgTimeMean
		res.MsgDelTimeStd = subTimeRunResults.MsgTimeStd

		totals.Successes += res.Successes
		totals.Failures += res.Failures
		totals.TotalMsgsPerSec += res.MsgsPerSec

		if res.MsgTimeMin < totals.MsgTimeMin {
			totals.MsgTimeMin = res.MsgTimeMin
		}

		if res.MsgTimeMax > totals.MsgTimeMax {
			totals.MsgTimeMax = res.MsgTimeMax
		}

		if subTimeRunResults.MsgTimeMin < totals.MsgDelTimeMin {
			totals.MsgDelTimeMin = subTimeRunResults.MsgTimeMin
		}

		if subTimeRunResults.MsgTimeMax > totals.MsgDelTimeMax {
			totals.MsgDelTimeMax = subTimeRunResults.MsgTimeMax
		}

		msgTimeMeansDelivered[i] = subTimeRunResults.MsgTimeMean
		msgTimeMeans[i] = res.MsgTimeMean
		msgsPerSecs[i] = res.MsgsPerSec
		runTimes[i] = res.RunTime
		bws[i] = res.MsgsPerSec
	}
	totals.Ratio = float64(totals.Successes) / float64(totals.Successes+totals.Failures)
	totals.AvgMsgsPerSec = stat.Mean(msgsPerSecs, nil)
	totals.AvgRunTime = stat.Mean(runTimes, nil)
	totals.MsgDelTimeMeanAvg = stat.Mean(msgTimeMeansDelivered, nil)
	totals.MsgDelTimeMeanStd = stat.StdDev(msgTimeMeansDelivered, nil)
	totals.MsgTimeMeanAvg = stat.Mean(msgTimeMeans, nil)
	totals.MsgTimeMeanStd = stat.StdDev(msgTimeMeans, nil)

	return totals
}

func printResults(results []*res.RunResults, totals *res.TotalResults, format string, quiet bool) {
	switch format {
	case "json":
		jr := JSONResults{
			Runs:   results,
			Totals: totals,
		}
		data, err := json.Marshal(jr)
		if err != nil {
			log.Printf("Failed to prepare results for printing - %s", err.Error())
		}
		var out bytes.Buffer
		json.Indent(&out, data, "", "\t")

		fmt.Println(string(out.Bytes()))
	default:
		if !quiet {
			for _, res := range results {
				fmt.Printf("======= CLIENT %s =======\n", res.ID)
				fmt.Printf("Ratio:               %.3f (%d/%d)\n", float64(res.Successes)/float64(res.Successes+res.Failures), res.Successes, res.Successes+res.Failures)
				fmt.Printf("Runtime (s):         %.3f\n", res.RunTime)
				fmt.Printf("Msg time min (us):   %.3f\n", res.MsgTimeMin)
				fmt.Printf("Msg time max (us):   %.3f\n", res.MsgTimeMax)
				fmt.Printf("Msg time mean (us):  %.3f\n", res.MsgTimeMean)
				fmt.Printf("Msg time std (us):   %.3f\n", res.MsgTimeStd)

				fmt.Printf("Bandwidth (msg/sec): %.3f\n\n", res.MsgsPerSec)
			}
		}
		fmt.Printf("========= TOTAL (%d) =========\n", len(results))
		fmt.Printf("Total Ratio:                 %.3f (%d/%d)\n", totals.Ratio, totals.Successes, totals.Successes+totals.Failures)
		fmt.Printf("Total Runtime (sec):         %.3f\n", totals.TotalRunTime)
		fmt.Printf("Average Runtime (sec):       %.3f\n", totals.AvgRunTime)
		fmt.Printf("Msg time min (us):           %.3f\n", totals.MsgTimeMin)
		fmt.Printf("Msg time max (us):           %.3f\n", totals.MsgTimeMax)
		fmt.Printf("Msg time mean mean (us):     %.3f\n", totals.MsgTimeMeanAvg)
		fmt.Printf("Msg time mean std (us):      %.3f\n", totals.MsgTimeMeanStd)

		fmt.Printf("Average Bandwidth (msg/sec): %.3f\n", totals.AvgMsgsPerSec)
		fmt.Printf("Total Bandwidth (msg/sec):   %.3f\n", totals.TotalMsgsPerSec)
	}
	return
}

func getTestTopic(channelID string) string {
	return "channels/" + channelID + "/messages/test"
}

func loadChansConfig(path *string, conns *Connections) {

	if _, err := toml.DecodeFile(*path, conns); err != nil {
		log.Fatalf("cannot load channels config %s \nuse tools/provision to create file", *path)
	}
}
