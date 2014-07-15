package reporter

import (
	"net"
	"snmp_poller/db_handler"
	"snmp_poller/netutils"
	"strconv"
	"strings"
	"time"
)

type QueueStat struct {
	Hostname string
	Ifindex  string
	QueueNum string
	Counter  int64
	Action   string
}

func GenerateQueueMsg(intf_name string, QStat QueueStat) []byte {
	time := strconv.FormatInt(time.Now().Unix(), 10)
	msg := strings.Join([]string{strings.Join([]string{"snmp.5m.queue", QStat.Hostname, intf_name,
		QStat.Action, QStat.QueueNum}, "."), strconv.FormatInt(QStat.Counter, 10), time, "\n"}, " ")
	return []byte(msg)
}

func QstatReporter(reporter_chan chan QueueStat,
	db_chan chan db_handler.InterfaceInfo,
	name chan string) {
	var interface_info db_handler.InterfaceInfo
	write_chan := make(chan []byte)
	feedback_chan := make(chan int)
	var reporterAddr net.TCPAddr
	reporterAddr.IP = net.ParseIP("127.0.0.1")
	reporterAddr.Port, _ = strconv.Atoi("2003")
	sock, err := net.DialTCP("tcp", nil, &reporterAddr)
	if err != nil {
		panic("cant connect to graphite server")
	}
	go netutils.WriteToTCP(sock, write_chan, feedback_chan)

	for {
		QStat := <-reporter_chan
		interface_info.Hostname = QStat.Hostname
		interface_info.Ifindex = QStat.Ifindex
		db_chan <- interface_info
		intf_name := <-name
		select {
		case <-feedback_chan:
			feedback_chan <- 1
			go netutils.ReconnectTCPW(reporterAddr, write_chan, feedback_chan)
		default:
			write_chan <- GenerateQueueMsg(intf_name, QStat)
		}

	}
}
