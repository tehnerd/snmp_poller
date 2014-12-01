package queue_stats

import (
	"fmt"

	"encoding/json"
	"github.com/soniah/gosnmp"
	"regexp"
	"snmp_poller/cfg"
	"snmp_poller/reporter"
	"strings"
	"time"
)

func SNMPPoll(RDescr cfg.RouterDescr, sync chan int, reporter_chan chan reporter.QueueStat,
	timeout int, retries int, redisChan chan reporter.QueueMsg) {
	var TargetRouter gosnmp.GoSNMP
	TargetRouter.Port = 161
	TargetRouter.Community = RDescr.Community
	TargetRouter.Version = gosnmp.Version2c
	TargetRouter.Timeout = time.Duration(timeout) * time.Second
	TargetRouter.Retries = retries
	TargetRouter.Target = RDescr.Name
	TargetRouter.MaxRepetitions = 30
	//	TargetRouter.Logger = log.New(os.Stdout, "", 0)
	err := TargetRouter.Connect()
	if err != nil {
		sync <- 1
		return
	}
	defer TargetRouter.Conn.Close()
	switch RDescr.Vendor {
	case "Huawei":
		resp, err := TargetRouter.BulkWalkAll(".1.3.6.1.4.1.2011.5.25.32.4.1.4.3.3.1")
		if err != nil {
			fmt.Println(err)
			if len(resp) == 0 {
				sync <- 1
				return
			}
		}
		QueueStatsHuawei(resp, reporter_chan, RDescr.Name, "Huawei", redisChan)
		sync <- 1

	case "Juniper":
		resp, err := TargetRouter.BulkWalkAll(".1.3.6.1.4.1.2636.3.15.1.1")
		if err != nil {
			fmt.Println(err)
			if len(resp) == 0 {
				sync <- 1
				return
			}
		}
		QueueStatsJuniper(resp, reporter_chan, RDescr.Name, "Juniper", redisChan)
		sync <- 1

	case "Cisco":
		resp, err := TargetRouter.BulkWalkAll("1.3.6.1.4.1.9.9.166.1.15.1.1")
		if err != nil {
			fmt.Println(err)
			if len(resp) == 0 {
				sync <- 1
				return
			}
		}
		QueueStatsCisco(resp, reporter_chan, RDescr.Name, "Cisco", redisChan)
		sync <- 1

	}
}

func QueueStatsHuawei(response []gosnmp.SnmpPDU, reporter_chan chan reporter.QueueStat,
	Hostname string, Vendor string, redisChan chan reporter.QueueMsg) {
	pass_re, _ := regexp.Compile(`^5.(\d+).0.(\d+)$`)
	drop_re, _ := regexp.Compile(`^9.(\d+).0.(\d+)$`)
	var QStat reporter.QueueStat
	QStat.Vendor = Vendor
	QStat.Hostname = Hostname
	ReportDict := make(map[string]map[string]int64)

	for cntr := 0; cntr < len(response); cntr++ {
		composite_oid := strings.Split(response[cntr].Name, ".1.3.6.1.4.1.2011.5.25.32.4.1.4.3.3.1.")
		if len(composite_oid) > 1 {
			oid := composite_oid[1]
			if len(pass_re.FindAllStringSubmatch(oid, -1)) != 0 {
				match := pass_re.FindAllStringSubmatch(oid, -1)
				ifindex := match[0][1]
				queue_num := match[0][2]
				queue_counter, ok := response[cntr].Value.(int64)
				if ok {
					if queue_counter != 0 {
						if _, exist := ReportDict[ifindex]; exist {
							ReportDict[ifindex][queue_num] = queue_counter
						} else {
							ReportDict[ifindex] = make(map[string]int64)
							ReportDict[ifindex][queue_num] = queue_counter
						}
						QStat.Ifindex = ifindex
						QStat.QueueNum = queue_num
						QStat.Counter = queue_counter
						QStat.Action = "pass"
						reporter_chan <- QStat
					}
				}
			} else if len(drop_re.FindAllStringSubmatch(oid, -1)) != 0 {
				match := drop_re.FindAllStringSubmatch(oid, -1)
				ifindex := match[0][1]
				queue_num := match[0][2]
				queue_counter, ok := response[cntr].Value.(int64)
				if ok {
					if queue_counter != 0 {
						QStat.Ifindex = ifindex
						QStat.QueueNum = queue_num
						QStat.Counter = queue_counter
						QStat.Action = "drop"
						reporter_chan <- QStat
					}
				}
			}

		}
	}
	JsonData, err := json.Marshal(ReportDict)
	if err == nil {
		var Msg reporter.QueueMsg
		Msg.RouterName = Hostname
		Msg.Data = JsonData
		redisChan <- Msg
	}
}

func QueueStatsJuniper(response []gosnmp.SnmpPDU, reporter_chan chan reporter.QueueStat,
	Hostname string, Vendor string, redisChan chan reporter.QueueMsg) {
	pass_re, _ := regexp.Compile(`^9.(\d+).(.+)$`)
	drop_re, _ := regexp.Compile(`^23.(\d+).(.+)$`)
	var QStat reporter.QueueStat
	QStat.Hostname = Hostname
	QStat.Vendor = Vendor
	ReportDict := make(map[string]map[string]int64)
	var DropCntr int64

	for cntr := 0; cntr < len(response); cntr++ {
		composite_oid := strings.Split(response[cntr].Name, ".1.3.6.1.4.1.2636.3.15.1.1.")
		if len(composite_oid) > 1 {
			oid := composite_oid[1]
			if len(pass_re.FindAllStringSubmatch(oid, -1)) != 0 {
				match := pass_re.FindAllStringSubmatch(oid, -1)
				ifindex := match[0][1]
				queue_num := match[0][2]
				queue_counter, ok := response[cntr].Value.(int64)
				if ok {
					if queue_counter != 0 {
						if _, exist := ReportDict[ifindex]; exist {
							ReportDict[ifindex][queue_num] = queue_counter
						} else {
							ReportDict[ifindex] = make(map[string]int64)
							ReportDict[ifindex][queue_num] = queue_counter
						}
						QStat.Ifindex = ifindex
						QStat.QueueNum = queue_num
						QStat.Counter = queue_counter
						QStat.Action = "pass"
						reporter_chan <- QStat
					}
				}
			} else if len(drop_re.FindAllStringSubmatch(oid, -1)) != 0 {
				match := drop_re.FindAllStringSubmatch(oid, -1)
				ifindex := match[0][1]
				queue_num := match[0][2]
				queue_counter, ok := response[cntr].Value.(int64)
				if ok {
					if queue_counter != 0 {
						QStat.Ifindex = ifindex
						QStat.QueueNum = queue_num
						QStat.Counter = queue_counter
						QStat.Action = "drop"
						reporter_chan <- QStat
						DropCntr += queue_counter
					}
				}
			}

		}
	}
	JsonData, err := json.Marshal(ReportDict)
	if err == nil {
		var Msg reporter.QueueMsg
		Msg.RouterName = Hostname
		Msg.Data = JsonData
		redisChan <- Msg
	}
	JsonData, err = json.Marshal(DropCntr)
	if err == nil {
		var Msg reporter.QueueMsg
		Hostname = strings.Join([]string{Hostname, "hm"}, "-")
		Msg.RouterName = Hostname
		Msg.Data = JsonData
		redisChan <- Msg
	}

}

func QueueStatsCisco(response []gosnmp.SnmpPDU, reporter_chan chan reporter.QueueStat,
	Hostname string, Vendor string, redisChan chan reporter.QueueMsg) {
	pass_re, _ := regexp.Compile(`^10.(\d+.\d+)$`)
	drop_re, _ := regexp.Compile(`^17.(\d+.\d+)$`)
	var QStat reporter.QueueStat
	QStat.Hostname = Hostname
	QStat.Vendor = Vendor
	ReportDict := make(map[string]map[string]int64)
	var DropCntr int64

	for cntr := 0; cntr < len(response); cntr++ {
		composite_oid := strings.Split(response[cntr].Name, "1.3.6.1.4.1.9.9.166.1.15.1.1.")
		if len(composite_oid) > 1 {
			oid := composite_oid[1]
			if len(pass_re.FindAllStringSubmatch(oid, -1)) != 0 {
				match := pass_re.FindAllStringSubmatch(oid, -1)
				composite_ifindex := match[0][1]
				queue_counter, ok := response[cntr].Value.(int64)
				if ok {
					if queue_counter != 0 {
						if _, exist := ReportDict[composite_ifindex]; exist {
							ReportDict[composite_ifindex]["1"] = queue_counter
						} else {
							ReportDict[composite_ifindex] = make(map[string]int64)
							ReportDict[composite_ifindex]["1"] = queue_counter
						}
						QStat.Ifindex = composite_ifindex
						QStat.QueueNum = "1"
						QStat.Counter = queue_counter
						QStat.Action = "pass"
						reporter_chan <- QStat
					}
				}
			} else if len(drop_re.FindAllStringSubmatch(oid, -1)) != 0 {
				match := drop_re.FindAllStringSubmatch(oid, -1)
				composite_ifindex := match[0][1]
				queue_counter, ok := response[cntr].Value.(int64)
				if ok {
					if queue_counter != 0 {
						QStat.Ifindex = composite_ifindex
						QStat.QueueNum = "1"
						QStat.Counter = queue_counter
						QStat.Action = "drop"
						reporter_chan <- QStat
						DropCntr += queue_counter
					}
				}
			}

		}
	}
	JsonData, err := json.Marshal(ReportDict)
	if err == nil {
		var Msg reporter.QueueMsg
		Msg.RouterName = Hostname
		Msg.Data = JsonData
		redisChan <- Msg
	}
	JsonData, err = json.Marshal(DropCntr)
	if err == nil {
		var Msg reporter.QueueMsg
		Hostname = strings.Join([]string{Hostname, "hm"}, "-")
		Msg.RouterName = Hostname
		Msg.Data = JsonData
		redisChan <- Msg
	}

}
