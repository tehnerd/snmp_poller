package main
import (
    "github.com/tehnerd/gosnmp"
    "fmt"
    "bufio"
    "os"
    "strings"
)

type RouterDescr struct {
    Name string
    Community string
    Vendor string
}


func ReadConfig() []RouterDescr {
    var rlist []RouterDescr
    fd,err := os.Open(os.Args[1])
    cfg_reader := bufio.NewReader(fd)
    line, err := cfg_reader.ReadString('\n')
    for err == nil{
        fields := strings.Fields(line)
        rlist = append(rlist,RouterDescr{fields[0],fields[1],fields[2]})
        line, err = cfg_reader.ReadString('\n')
    }
    return rlist
}

func snmp_pull(RDescr RouterDescr,sync chan int){
    s,_ := gosnmp.NewGoSNMP(RDescr.Name,RDescr.Community,gosnmp.Version2c,20)
    resp,_ := s.BulkWalk(30,".1.3.6.1.4.1.2011.5.25.32.4.1.4.3.3.1.")
    fmt.Println(len(resp))
    sync <- 1
}

func main(){
    rlist := ReadConfig()
    sync := make(chan int)
    for cntr:=0;cntr<len(rlist);cntr++{
        go snmp_pull(rlist[cntr],sync)
    }
    for cntr:=0;cntr<len(rlist);cntr++{
        <- sync
    }
}
