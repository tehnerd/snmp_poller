package main
import (
    "github.com/tehnerd/gosnmp"
    "fmt"
    "bufio"
    "os"
    "strings"
    "strconv"
)

type RouterDescr struct {
    Name string
    Community string
    Vendor string
}


func ReadConfig() ([]RouterDescr,[]string,int) {
    var rlist []RouterDescr
    var tasks []string
    pullers := 10
    fd,err := os.Open(os.Args[1])
    defer fd.Close()
    cfg_reader := bufio.NewReader(fd)
    line, err := cfg_reader.ReadString('\n')
    for err == nil{
        fields := strings.Fields(line)
        if fields[0] == "tasks:"{
            for cntr:=1;cntr<len(fields);cntr++{
                tasks = append(tasks,fields[cntr])
            }
        } else if fields[0] == "pullers:"{
            pullers,_ = strconv.Atoi(fields[1])
        } else {
            rlist = append(rlist,RouterDescr{fields[0],fields[1],fields[2]})
        }    
        line, err = cfg_reader.ReadString('\n')
    }
    return rlist, tasks, pullers
}

func snmp_pull(RDescr RouterDescr,sync chan int){
    s,_ := gosnmp.NewGoSNMP(RDescr.Name,RDescr.Community,gosnmp.Version2c,20)
    resp,_ := s.BulkWalk(30,".1.3.6.1.4.1.2011.5.25.32.4.1.4.3.3.1.")
    fmt.Println(len(resp))
    sync <- 1
}

func main(){
    if len(os.Args) < 2{
        os.Exit(1)
    }
    rlist,tasks, MAX_PULLERS := ReadConfig()
    fmt.Println(tasks)
    fmt.Println(MAX_PULLERS)
    sync := make(chan int)
    running_pullers := 0
    for cntr:=0;cntr<len(rlist);{
        if running_pullers < MAX_PULLERS {
            go snmp_pull(rlist[cntr],sync)
            cntr++
            running_pullers += 1
        } else 
        {
            <- sync
            running_pullers -= 1
        }
    }
    for cntr:=0;cntr<running_pullers;cntr++{
        <- sync
    }
}
