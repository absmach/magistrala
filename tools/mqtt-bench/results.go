// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"time"

	mat "gonum.org/v1/gonum/mat"
	stat "gonum.org/v1/gonum/stat"
)

type runResults struct {
	ID             string  `json:"id"`
	Successes      int64   `json:"successes"`
	Failures       int64   `json:"failures"`
	RunTime        float64 `json:"run_time"`
	MsgTimeMin     float64 `json:"msg_time_min"`
	MsgTimeMax     float64 `json:"msg_time_max"`
	MsgTimeMean    float64 `json:"msg_time_mean"`
	MsgTimeStd     float64 `json:"msg_time_std"`
	MsgDelTimeMin  float64 `json:"msg_del_time_min"`
	MsgDelTimeMax  float64 `json:"msg_del_time_max"`
	MsgDelTimeMean float64 `json:"msg_del_time_mean"`
	MsgDelTimeStd  float64 `json:"msg_del_time_std"`
	MsgsPerSec     float64 `json:"msgs_per_sec"`
}

type subsResults map[string](*[]float64)

type totalResults struct {
	Ratio             float64 `json:"ratio"`
	Successes         int64   `json:"successes"`
	Failures          int64   `json:"failures"`
	TotalRunTime      float64 `json:"total_run_time"`
	AvgRunTime        float64 `json:"avg_run_time"`
	MsgTimeMin        float64 `json:"msg_time_min"`
	MsgTimeMax        float64 `json:"msg_time_max"`
	MsgDelTimeMin     float64 `json:"msg_del_time_min"`
	MsgDelTimeMax     float64 `json:"msg_del_time_max"`
	MsgTimeMeanAvg    float64 `json:"msg_time_mean_avg"`
	MsgTimeMeanStd    float64 `json:"msg_time_mean_std"`
	MsgDelTimeMeanAvg float64 `json:"msg_del_time_mean_avg"`
	MsgDelTimeMeanStd float64 `json:"msg_del_time_mean_std"`
	TotalMsgsPerSec   float64 `json:"total_msgs_per_sec"`
	AvgMsgsPerSec     float64 `json:"avg_msgs_per_sec"`
}

func calculateTotalResults(results []*runResults, totalTime time.Duration, sr subsResults) *totalResults {
	if results == nil || len(results) < 1 {
		return nil
	}
	totals := new(totalResults)
	msgTimeMeans := make([]float64, len(results))
	msgTimeMeansDelivered := make([]float64, len(results))
	msgsPerSecs := make([]float64, len(results))
	runTimes := make([]float64, len(results))
	bws := make([]float64, len(results))

	totals.TotalRunTime = totalTime.Seconds()

	totals.MsgTimeMin = results[0].MsgTimeMin
	for i, res := range results {

		totals.Successes += res.Successes
		totals.Failures += res.Failures
		totals.TotalMsgsPerSec += res.MsgsPerSec

		if res.MsgTimeMin < totals.MsgTimeMin {
			totals.MsgTimeMin = res.MsgTimeMin
		}

		if res.MsgTimeMax > totals.MsgTimeMax {
			totals.MsgTimeMax = res.MsgTimeMax
		}

		if res.MsgDelTimeMin < totals.MsgDelTimeMin {
			totals.MsgDelTimeMin = res.MsgDelTimeMin
		}

		if res.MsgDelTimeMax > totals.MsgDelTimeMax {
			totals.MsgDelTimeMax = res.MsgDelTimeMax
		}

		msgTimeMeansDelivered[i] = res.MsgDelTimeMean
		msgTimeMeans[i] = res.MsgTimeMean
		msgsPerSecs[i] = res.MsgsPerSec
		runTimes[i] = res.RunTime
		bws[i] = res.MsgsPerSec
	}

	for _, v := range sr {
		times := mat.NewDense(1, len(*v), *v)
		totals.MsgDelTimeMin = mat.Min(times) / 1000
		totals.MsgDelTimeMax = mat.Max(times) / 1000
		totals.MsgDelTimeMeanAvg = stat.Mean(*v, nil) / 1000
		totals.MsgDelTimeMeanStd = stat.StdDev(*v, nil) / 1000
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

func printResults(results []*runResults, totals *totalResults, format string, quiet bool) {
	switch format {
	case "json":
		jr := JSONResults{
			Runs:   results,
			Totals: totals,
		}
		data, err := json.Marshal(jr)
		if err != nil {
			log.Printf("Failed to prepare results for printing - %s\n", err.Error())
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
				fmt.Printf("Msg time std (us):   %.3f\n\n", res.MsgTimeStd)

				fmt.Printf("Msg del time min (us):   %.3f\n", res.MsgDelTimeMin)
				fmt.Printf("Msg del time max (us):   %.3f\n", res.MsgDelTimeMax)
				fmt.Printf("Msg del time mean (us):  %.3f\n", res.MsgDelTimeMean)
				fmt.Printf("Msg del time std (us):   %.3f\n", res.MsgDelTimeStd)

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

		fmt.Printf("Msg del time min (us):   %.3f\n", totals.MsgDelTimeMin)
		fmt.Printf("Msg del time max (us):   %.3f\n", totals.MsgDelTimeMax)
		fmt.Printf("Msg del time mean (us):  %.3f\n", totals.MsgDelTimeMeanAvg)
		fmt.Printf("Msg del time std (us):   %.3f\n", totals.MsgDelTimeMeanStd)

		fmt.Printf("Average Bandwidth (msg/sec): %.3f\n", totals.AvgMsgsPerSec)
		fmt.Printf("Total Bandwidth (msg/sec):   %.3f\n", totals.TotalMsgsPerSec)
	}
	return
}
