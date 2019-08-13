package res

// RunResults describes results of a single client / run
type RunResults struct {
	ID        string `json:"id"`
	Successes int64  `json:"successes"`
	Failures  int64  `json:"failures"`

	RunTime     float64 `json:"run_time"`
	MsgTimeMin  float64 `json:"msg_time_min"`
	MsgTimeMax  float64 `json:"msg_time_max"`
	MsgTimeMean float64 `json:"msg_time_mean"`
	MsgTimeStd  float64 `json:"msg_time_std"`

	MsgDelTimeMin  float64 `json:"msg_del_time_min"`
	MsgDelTimeMax  float64 `json:"msg_del_time_max"`
	MsgDelTimeMean float64 `json:"msg_del_time_mean"`
	MsgDelTimeStd  float64 `json:"msg_del_time_std"`

	MsgsPerSec float64 `json:"msgs_per_sec"`
}

// SubTimes - measuring time of arrival of message in subs
type SubTimes map[string][]float64

// TotalResults describes results of all clients / runs
type TotalResults struct {
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
