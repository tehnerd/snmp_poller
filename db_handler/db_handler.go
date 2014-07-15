package db_handler

import (
	"fmt"
	"strings"

	"github.com/mxk/go-sqlite/sqlite3"
)

type InterfaceInfo struct {
	Hostname string
	Ifindex  string
}

func GetInterfaceNameSQLite(interface_info chan InterfaceInfo, db_location string,
	name chan string) {
	InterfaceDict := make(map[string]map[string]string)
	db_conn, err := sqlite3.Open(db_location)
	var intf_name string
	defer db_conn.Close()
	if err != nil {
		fmt.Println(err)
		panic("cant open db")
	}
	sql := "select intfname from ifindex where rname=? and ifindex=?"
	for {
		select {
		case intf := <-interface_info:
			hostname_dict, dict_exist := InterfaceDict[intf.Hostname]
			if dict_exist {
				intf_name, intf_exist := hostname_dict[intf.Ifindex]
				if intf_exist {
					name <- intf_name
					continue
				}
			} else {
				fmt.Println("not exist")
				InterfaceDict[intf.Hostname] = make(map[string]string)
			}
			query, err := db_conn.Query(sql, intf.Hostname, intf.Ifindex)
			if err != nil {
				fmt.Println(err)
				name <- "NaN"
				continue
			}
			query.Scan(&intf_name)
			if len(intf_name) == 0 {
				name <- "NaN"
				continue
			}
			intf_name = strings.Replace(intf_name, "/", "-", -1)
			intf_name = strings.Replace(intf_name, ".", "-", -1)
			InterfaceDict[intf.Hostname][intf.Ifindex] = intf_name
			name <- intf_name
		}
	}
}
