package main

import (
	"bufio"
	"os"
	"snmp_poller/cfg"
	"snmp_poller/db_handler"
	"snmp_poller/queue_stats"
	"snmp_poller/reporter"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

func ReadConfig() ([]cfg.RouterDescr, cfg.GenericCfg) {
	var rlist []cfg.RouterDescr
	var Cfg cfg.GenericCfg
	Cfg.Pollers = 10
	Cfg.Timeout = 10
	Cfg.Retries = 3
	Cfg.GraphiteIP = "127.0.0.1"
	Cfg.RedisAddr = "127.0.0.1:6379"
	fd, err := os.Open(os.Args[1])
	defer fd.Close()
	cfg_reader := bufio.NewReader(fd)
	line, err := cfg_reader.ReadString('\n')
	for err == nil {
		fields := strings.Fields(line)
		if fields[0] == "tasks:" {
			for cntr := 1; cntr < len(fields); cntr++ {
				Cfg.Tasks = append(Cfg.Tasks, fields[cntr])
			}
		} else if fields[0] == "pollers:" {
			Cfg.Pollers, _ = strconv.Atoi(fields[1])
		} else if fields[0] == "timeout:" {
			Cfg.Timeout, _ = strconv.Atoi(fields[1])
		} else if fields[0] == "retries:" {
			Cfg.Retries, _ = strconv.Atoi(fields[1])
		} else if fields[0] == "graphite:" {
			Cfg.GraphiteIP = fields[1]
		} else if fields[0] == "redis:" {
			Cfg.RedisAddr = fields[1]
		} else {
			rlist = append(rlist, cfg.RouterDescr{fields[0], fields[1], fields[2]})
		}
		line, err = cfg_reader.ReadString('\n')
	}
	return rlist, Cfg
}

func StartPolling(rlist []cfg.RouterDescr, MAX_POLLERS int,
	reporter_chan chan reporter.QueueStat,
	timeout int, retries int, sync_flag *int32,
	redisChan chan reporter.QueueMsg) {
	atomic.AddInt32(sync_flag, 1)
	running_pollers := 0
	sync := make(chan int)
	for cntr := 0; cntr < len(rlist); {
		if running_pollers < MAX_POLLERS {
			go queue_stats.SNMPPoll(rlist[cntr], sync, reporter_chan,
				timeout, retries, redisChan)
			cntr++
			running_pollers += 1
		} else {
			<-sync
			running_pollers -= 1
		}
	}
	for cntr := 0; cntr < running_pollers; cntr++ {
		<-sync
	}
	atomic.AddInt32(sync_flag, -1)

}

func main() {
	if len(os.Args) < 3 {
		os.Exit(1)
	}
	rlist, Cfg := ReadConfig()
	reporter_chan := make(chan reporter.QueueStat)
	db_chan := make(chan db_handler.InterfaceInfo)
	redisChan := make(chan reporter.QueueMsg)
	go reporter.SendToRedis(Cfg.RedisAddr, redisChan)
	name_chan := make(chan string)
	sync_flag := int32(0)
	go db_handler.GetInterfaceNameSQLite(db_chan, os.Args[2], name_chan)
	go reporter.QstatReporter(reporter_chan, db_chan, name_chan, Cfg.GraphiteIP)
	for {
		//protecting ourself against dead reporter(wont run unlim ammount of
		//polling jobs, coz workers cant report anyway and will hang at report_chan <-
		if !atomic.CompareAndSwapInt32(&sync_flag, 3, 3) {
			go StartPolling(rlist, Cfg.Pollers, reporter_chan, Cfg.Timeout, Cfg.Retries,
				&sync_flag, redisChan)
		}
		time.Sleep(5 * time.Minute)
	}
}
