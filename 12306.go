package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	"github.com/horsley/svrkit"
)

//R 全局入口
var R railway12306

const (
	stationShenZhenBei  = "IOQ"
	stationGuangZhouNan = "IZQ"
	stationNanTou       = "NOQ"
	stationRongGui      = "RUQ"
)

const (
	apiTicketQuery = "https://kyfw.12306.cn/otn/leftTicket/queryZ?leftTicketDTO.train_date=%s&leftTicketDTO.from_station=%s&leftTicketDTO.to_station=%s&purpose_codes=ADULT"
)

type railway12306 struct {
	//代码 -> 名字
	stationCode2NameMap map[string]string

	//名字 -> 代码
	stationName2CodeMap map[string]string
}

type apiResponse struct {
	Data       interface{}
	HTTPStatus int
	Message    string
	Status     bool
}

type ticketQueryResponse struct {
	Data struct {
		Flag   string
		Map    map[string]string
		Result []string
	}
	apiResponse
}

type lineInfo []string

func (l lineInfo) String() string {
	from := R.GetStationNameByCode(l[lineInfoFromStationTelecode])
	to := R.GetStationNameByCode(l[lineInfoToStationTelecode])

	startTime := l[lineInfoStartTime]
	arrivalTime := l[lineInfoArriveTime]
	return fmt.Sprintf("%s %s(%s) -> %s(%s) 二等座:%s",
		l[lineInfoStationTrainCode], from, startTime, to, arrivalTime,
		l[lineInfoSeatGrade2])
}

type lineInfoFieldIndex int

const (
	lineInfoSecret lineInfoFieldIndex = iota
	lineInfoBtnText
	lineInfoTrainNo              //车票号 2
	lineInfoStationTrainCode     //车次 3
	lineInfoStartStationTelecode //始发站代号 4
	lineInfoEndStationTelecode   //终点站代号 5
	lineInfoFromStationTelecode  //出发站代号 6
	lineInfoToStationTelecode    //到达站代号 7
	lineInfoStartTime            //出发时间 8
	lineInfoArriveTime           //到达时间 9
	lineInfoLishi                //历时 10
	lineInfoCanWebBuy            //能否购买 Y 可以 11
	_                            //12
	_                            //13
	_                            //14
	_                            //15
	_                            //16
	_                            //17
	_                            //18
	_                            //19
	_                            //20
	_                            //21
	_                            //22
	_                            //23 软卧
	_                            //24 软座
	_                            //25
	lineInfoSeatNoSeat           //26 无座
	_                            //27
	_                            //28 硬卧
	_                            //29
	lineInfoSeatGrade2           //30 二等座
	lineInfoSeatGrade1           //31 一等座
	_                            //32 商务特等座

)

func (r *railway12306) LoadStationNames(jsFile string) error {
	bin, err := ioutil.ReadFile(jsFile)
	if err != nil {
		return err
	}
	pieces := strings.Split(string(bin), "'")
	if len(pieces) != 3 {
		return errors.New("not recognize format")
	}

	stations := strings.Split(pieces[1], "@")

	r.stationCode2NameMap = make(map[string]string)
	r.stationName2CodeMap = make(map[string]string)

	for _, s := range stations {
		p := strings.Split(s, "|")
		if len(p) > 2 {
			r.stationName2CodeMap[p[1]] = p[2]
			r.stationCode2NameMap[p[2]] = p[1]
		}
	}

	return nil
}

func (r *railway12306) GetStationNameByCode(code string) string {
	if name, ok := r.stationCode2NameMap[code]; ok {
		return name
	}
	return code
}

func (r *railway12306) QueryTickets(date, from, to string) ([]lineInfo, error) {
	bin, err := svrkit.HTTPGet(fmt.Sprintf(apiTicketQuery, date, from, to))
	if err != nil {
		return nil, err
	}

	var resp ticketQueryResponse
	err = json.Unmarshal(bin, &resp)
	if err != nil {
		return nil, err
	}

	result := make([]lineInfo, len(resp.Data.Result))

	for i, v := range resp.Data.Result {
		result[i] = strings.Split(v, "|")
	}
	return result, nil
}

func (r *railway12306) FilterLinesCanNotBuy(lines []lineInfo) []lineInfo {
	result := make([]lineInfo, 0)
	for _, v := range lines {
		if v[lineInfoCanWebBuy] != "Y" {
			continue
		}
		result = append(result, v)
	}
	return result
}

func (r *railway12306) FilterLinesWithoutSeatGrade2(lines []lineInfo) []lineInfo {
	result := make([]lineInfo, 0)
	for _, v := range lines {
		if v[lineInfoSeatGrade2] == "无" {
			continue
		}

		if v[lineInfoSeatGrade2] != "有" {
			if svrkit.MustInt(v[lineInfoSeatGrade2]) < 2 {
				continue
			}
		}
		result = append(result, v)
	}
	return result
}

func (r *railway12306) FilterStationNotMatch(lines []lineInfo, from, to string) []lineInfo {
	result := make([]lineInfo, 0)
	for _, v := range lines {
		if v[lineInfoFromStationTelecode] != from || v[lineInfoToStationTelecode] != to {
			continue
		}
		result = append(result, v)
	}
	return result
}

//返回时间字符串 A - B 的差值，返回单位分钟
var timeStringCheckRegex = regexp.MustCompile(`[012][0-9]:[0-5][0-9]`)

func (r *railway12306) TimeGap(a, b string) int {
	return r.TimeString2Min(a) - r.TimeString2Min(b)
}

func (r *railway12306) TimeString2Min(timeStr string) int {
	if !timeStringCheckRegex.MatchString(timeStr) {
		log.Println("time string  invalid:", timeStr)
		return 0
	}
	// log.Println(int(timeStr[0]-'0')*10*60, int(timeStr[1]-'0')*60, int(timeStr[3]-'0')*10, int(timeStr[4]-'0'))
	return int(timeStr[0]-'0')*10*60 + int(timeStr[1]-'0')*60 + int(timeStr[3]-'0')*10 + int(timeStr[4]-'0')
}

type trainTransfer struct {
	first  lineInfo
	second lineInfo
}

func (t *trainTransfer) Gap() int {
	return R.TimeGap(t.second[lineInfoStartTime], t.first[lineInfoArriveTime])
}

func (t trainTransfer) String() string {
	return fmt.Sprintf("%s 换乘(间隔：%d min) %s", t.first, t.Gap(), t.second)
}


type roundTrip struct {
	first trainTransfer
	second trainTransfer
}