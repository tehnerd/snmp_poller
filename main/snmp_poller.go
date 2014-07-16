package main

import (
	"bufio"
	"fmt"
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

func ReadConfig() ([]cfg.RouterDescr, []string, int, int, int) {
	var rlist []cfg.RouterDescr
	var tasks []string
	pollers := 10
	timeout := 10
	retries := 3
	fd, err := os.Open(os.Args[1])
	defer fd.Close()
	cfg_reader := bufio.NewReader(fd)
	line, err := cfg_reader.ReadString('\n')
	for err == nil {
		fields := strings.Fields(line)
		if fields[0] == "tasks:" {
			for cntr := 1; cntr < len(fields); cntr++ {
				tasks = append(tasks, fields[cntr])
			}
		} else if fields[0] == "pollers:" {
			pollers, _ = strconv.Atoi(fields[1])
		} else if fields[0] == "timeout:" {
			timeout, _ = strconv.Atoi(fields[1])
		} else if fields[0] == "retries:" {
			retries, _ = strconv.Atoi(fields[1])

		} else {
			rlist = append(rlist, cfg.RouterDescr{fields[0], fields[1], fields[2]})
		}
		line, err = cfg_reader.ReadString('\n')
	}
	return rlist, tasks, pollers, timeout, retries
}

//TODO: add cfg struct instead of sep values
func StartPolling(rlist []cfg.RouterDescr, MAX_POLLERS int,
	reporter_chan chan reporter.QueueStat,
	timeout int, retries int, sync_flag *int32) {
	atomic.AddInt32(sync_flag, 1)
	running_pollers := 0
	sync := make(chan int)
	for cntr := 0; cntr < len(rlist); {
		if running_pollers < MAX_POLLERS {
			go queue_stats.SNMPPoll(rlist[cntr], sync, reporter_chan,
				timeout, retries)
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
	rlist, tasks, MAX_POLLERS, timeout, retries := ReadConfig()
	fmt.Println(tasks)
	fmt.Println(MAX_POLLERS)
	reporter_chan := make(chan reporter.QueueStat)
	db_chan := make(chan db_handler.InterfaceInfo)
	name_chan := make(chan string)
	sync_flag := int32(0)
	go db_handler.GetInterfaceNameSQLite(db_chan, os.Args[2], name_chan)
	go reporter.QstatReporter(reporter_chan, db_chan, name_chan)
	for {
		//protecting ourself against dead reporter(wont run unlim ammount of
		//polling jobs, coz workers cant report anyway and will hang and report_chan <-
		if !atomic.CompareAndSwapInt32(&sync_flag, 3, 3) {
			go StartPolling(rlist, MAX_POLLERS, reporter_chan, timeout, retries,
				&sync_flag)
		}
		time.Sleep(5 * time.Minute)
	}
}
