package db_handler

import (
	"strings"

	"github.com/mxk/go-sqlite/sqlite3"
)

type InterfaceInfo struct {
	Hostname string
	Ifindex  string
	QueueNum string
	InfoType string // we could ask for ifindex resolution or queue name
}

func GetInterfaceNameSQLite(interface_info chan InterfaceInfo, db_location string,
	name chan string) {
	InterfaceDict := make(map[string]map[string]string)
	db_conn, err := sqlite3.Open(db_location)
	var var_name string
	defer db_conn.Close()
	if err != nil {
		panic("cant open db")
	}
	sql_intf := "select intfname from ifindex where rname=? and ifindex=?"
	jqueue_name := "select queue_name from juniper_queue_name where queue_index=?"
	for {
		select {
		case intf := <-interface_info:
			hostname_dict, dict_exist := InterfaceDict[intf.Hostname]
			if dict_exist {
				switch intf.InfoType {
				case "intf":
					intf_name, intf_exist := hostname_dict[intf.Ifindex]
					if intf_exist {
						name <- intf_name
						continue
					}
				case "jqueue":
					intf_queue, queue_exist := hostname_dict[intf.QueueNum]
					if queue_exist {
						name <- intf_queue
						continue
					}

				}
			} else {
				InterfaceDict[intf.Hostname] = make(map[string]string)
			}
			query := new(sqlite3.Stmt)
			switch intf.InfoType {
			case "intf":
				query, err = db_conn.Query(sql_intf, intf.Hostname, intf.Ifindex)
				if err != nil {
					name <- "NaN"
					continue
				}
			case "jqueue":
				query, err = db_conn.Query(jqueue_name, intf.QueueNum)
				if err != nil {
					name <- "NaN"
					continue
				}
			}
			query.Scan(&var_name)
			if len(var_name) == 0 {
				name <- "NaN"
				continue
			}
			var_name = strings.Replace(var_name, "/", "-", -1)
			var_name = strings.Replace(var_name, ".", "-", -1)
			switch intf.InfoType {
			case "intf":
				InterfaceDict[intf.Hostname][intf.Ifindex] = var_name
			case "jqueue":
				InterfaceDict[intf.Hostname][intf.QueueNum] = var_name
			}
			name <- var_name
		}
	}
}
