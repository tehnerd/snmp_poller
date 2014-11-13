package reporter

import (
	"github.com/garyburd/redigo/redis"
	"net"
	"snmp_poller/db_handler"
	"snmp_poller/netutils"
	"strconv"
	"strings"
	"time"
)

type QueueStat struct {
	Vendor   string
	Hostname string
	Ifindex  string
	QueueNum string
	Counter  int64
	Action   string
}

func GenerateQueueMsg(QStat QueueStat) []byte {
	time := strconv.FormatInt(time.Now().Unix(), 10)
	msg := strings.Join([]string{strings.Join([]string{"snmp.5m.queue", QStat.Hostname, QStat.Ifindex,
		QStat.Action, QStat.QueueNum}, "."), strconv.FormatInt(QStat.Counter, 10), time, "\n"}, " ")
	return []byte(msg)
}

func QstatReporter(reporter_chan chan QueueStat,
	db_chan chan db_handler.InterfaceInfo,
	name chan string, graphite_ip string) {
	var interface_info db_handler.InterfaceInfo
	write_chan := make(chan []byte)
	feedback_chan := make(chan int)
	var reporterAddr net.TCPAddr
	reporterAddr.IP = net.ParseIP(graphite_ip)
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
		interface_info.InfoType = "intf"
		db_chan <- interface_info
		QStat.Ifindex = <-name
		if QStat.Vendor == "Juniper" {
			interface_info.InfoType = "jqueue"
			interface_info.QueueNum = QStat.QueueNum
			db_chan <- interface_info
			QStat.QueueNum = <-name
		}
		select {
		case <-feedback_chan:
			feedback_chan <- 1
			go netutils.ReconnectTCPW(reporterAddr, write_chan, feedback_chan)
		default:
			write_chan <- GenerateQueueMsg(QStat)
		}

	}
}

type QueueMsg struct {
	RouterName string
	Data       []byte
}

//TODO: add logic for reconnection
func SendToRedis(redisAddress string, dataChan chan QueueMsg) {
	redisConnection, err := redis.Dial("tcp", redisAddress)
	if err != nil {
		panic("cant connect to redis server")
	}
	for {
		select {
		case msg := <-dataChan:
			oldData, _ := redis.Bytes(redisConnection.Do("GET", msg.RouterName))
			redisConnection.Do("SET", strings.Join([]string{msg.RouterName, "-old"}, ""), oldData)
			redisConnection.Do("SET", msg.RouterName, msg.Data)
		}
	}
}
