package main

import (
	"encoding/json"
	"time"

	"flag"
	"fmt"
	"net"
	"net/rpc"
	"strconv"
	"strings"
	//repush
	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/log"
	rpch "github.com/tidusant/chadmin-repo/cuahang"
	"github.com/tidusant/chadmin-repo/models"
)

const (
	defaultcampaigncode string = "XVsdAZGVmY"
)

type Arith int

func (t *Arith) Run(data string, result *string) error {
	log.Debugf("Call RPC orders args:" + data)
	*result = ""
	//parse args
	args := strings.Split(data, "|")

	if len(args) < 3 {
		return nil
	}
	var usex models.UserSession
	usex.Session = args[0]
	usex.Action = args[2]
	info := strings.Split(args[1], "[+]")
	usex.UserID = info[0]
	ShopID := info[1]
	usex.Params = ""
	if len(args) > 3 {
		usex.Params = args[3]
	}

	//check shop permission
	shop := rpch.GetShopById(usex.UserID, ShopID)
	if shop.Status == 0 {
		*result = c3mcommon.ReturnJsonMessage("-4", "Shop is disabled.", "", "")
		return nil
	}
	usex.Shop = shop

	if usex.Action == "la" {
		*result = LoadAll(usex)
	} else if usex.Action == "laa" {
		*result = LoadAllActive(usex)
	} else if usex.Action == "ld" {
		*result = LoadDetail(usex)
	} else if usex.Action == "sc" {
		*result = SaveCampaign(usex)
	} else if usex.Action == "dc" {
		*result = DeleteCampaign(usex)
	} else { //default
		*result = c3mcommon.ReturnJsonMessage("-5", "Action not found.", "", "")
	}

	return nil
}

func LoadAll(usex models.UserSession) string {

	//default status
	camps := rpch.GetAllCampaigns(usex.Shop.ID.Hex())
	for k, v := range camps {
		camps[k] = rpch.GetCampaignDetailByID(usex.Shop.ID.Hex(), v)
	}
	info, _ := json.Marshal(camps)
	strrt := string(info)
	return c3mcommon.ReturnJsonMessage("1", "", "success", strrt)
}
func LoadAllActive(usex models.UserSession) string {
	//default status
	t := time.Now()
	d, _ := time.ParseDuration(strconv.Itoa(t.Hour()-23) + "h" + strconv.Itoa(t.Minute()-59) + "m")
	camps := rpch.GetCampaignsByRange(usex.Shop.ID.Hex(), t.UTC().Add(d).AddDate(0, 0, -1), t.UTC().Add(d).AddDate(0, 0, 1))
	strrt := `[`

	for _, v := range camps {
		strrt += `{"ID":"` + v.ID.Hex() + `","Name":"` + v.Name + `"},`
	}
	if len(camps) > 0 {
		strrt = strrt[:len(strrt)-1]
	}
	strrt += `]`
	return c3mcommon.ReturnJsonMessage("1", "", "success", strrt)
}
func LoadDetail(usex models.UserSession) string {
	camps := rpch.GetCampaignByID(usex.Shop.ID.Hex(), usex.Params)

	campdetails := rpch.GetCampaignDetailByID(usex.Shop.ID.Hex(), camps)

	info, _ := json.Marshal(campdetails)
	strrt := string(info)
	return c3mcommon.ReturnJsonMessage("1", "", "success", strrt)
}

func SaveCampaign(usex models.UserSession) string {

	var camp models.Campaign
	err := json.Unmarshal([]byte(usex.Params), &camp)
	if !c3mcommon.CheckError("update status parse json", err) {
		return c3mcommon.ReturnJsonMessage("0", "update status fail", "", "")
	}
	//check old status
	oldcamp := camp
	if oldcamp.ID.Hex() != "" {
		oldcamp = rpch.GetCampaignByID(usex.Shop.ID.Hex(), camp.ID.Hex())
		oldcamp.Name = camp.Name
		oldcamp.Bugget = camp.Bugget
		oldcamp.Start = camp.Start
		oldcamp.End = camp.End
	} else {
		oldcamp.UserId = usex.UserID
		oldcamp.ShopId = usex.Shop.ID.Hex()
	}

	oldcamp = rpch.SaveCampaign(oldcamp)
	b, _ := json.Marshal(oldcamp)
	return c3mcommon.ReturnJsonMessage("1", "", "success", string(b))
}

func DeleteCampaign(usex models.UserSession) string {

	oldcamp := rpch.GetCampaignByID(usex.Shop.ID.Hex(), usex.Params)
	if oldcamp.ID.Hex() == "" {
		return c3mcommon.ReturnJsonMessage("-5", "Campaign not found.", "", "")
	}
	if oldcamp.Noo > 0 {
		return c3mcommon.ReturnJsonMessage("-5", "Campaign has "+strconv.Itoa(oldcamp.Noo)+" orders.", "", "")
	}

	rpch.DeleteCampaign(oldcamp)

	return c3mcommon.ReturnJsonMessage("1", "", "success", `"`+oldcamp.ID.Hex()+`"`)
}

func main() {
	var port int
	var debug bool
	flag.IntVar(&port, "port", 9885, "help message for flagname")
	flag.BoolVar(&debug, "debug", false, "Indicates if debug messages should be printed in log files")
	flag.Parse()

	logLevel := log.DebugLevel
	if !debug {
		logLevel = log.InfoLevel

	}

	log.SetOutputFile(fmt.Sprintf("adminCampaign-"+strconv.Itoa(port)), logLevel)
	defer log.CloseOutputFile()
	log.RedirectStdOut()

	//init db
	arith := new(Arith)
	rpc.Register(arith)
	log.Infof("running with port:" + strconv.Itoa(port))

	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(port))
	c3mcommon.CheckError("rpc dail:", err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	c3mcommon.CheckError("rpc init listen", err)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(conn)
	}
}
