package queue_stats

import (
	"regexp"
	"snmp_poller/cfg"
	"snmp_poller/reporter"
	"strings"

	"github.com/tehnerd/gosnmp"
)

func SNMPPoll(RDescr cfg.RouterDescr, sync chan int, reporter_chan chan reporter.QueueStat,
	timeout int) {
	s, _ := gosnmp.NewGoSNMP(RDescr.Name, RDescr.Community, gosnmp.Version2c, int64(timeout))
	switch RDescr.Vendor {
	case "Huawei":
		resp, err := s.BulkWalk(40, ".1.3.6.1.4.1.2011.5.25.32.4.1.4.3.3.1")
		if err != nil {
			sync <- 1
		}
		QueueStatsHuawei(resp, reporter_chan, RDescr.Name)
		sync <- 1
	}
}

func QueueStatsHuawei(response []gosnmp.SnmpPDU, reporter_chan chan reporter.QueueStat,
	Hostname string) {
	pass_re, _ := regexp.Compile(`^5.(\d+).0.(\d+)$`)
	drop_re, _ := regexp.Compile(`^9.(\d+).0.(\d+)$`)
	var QStat reporter.QueueStat
	QStat.Hostname = Hostname

	for cntr := 0; cntr < len(response); cntr++ {
		composite_oid := strings.Split(response[cntr].Name, "1.3.6.1.4.1.2011.5.25.32.4.1.4.3.3.1.")
		if len(composite_oid) > 1 {
			oid := composite_oid[1]
			if len(pass_re.FindAllStringSubmatch(oid, -1)) != 0 {
				match := pass_re.FindAllStringSubmatch(oid, -1)
				ifindex := match[0][1]
				queue_num := match[0][2]
				queue_counter, ok := response[cntr].Value.(int64)
				if ok {
					if queue_counter != 0 {
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
}
