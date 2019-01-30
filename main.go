package main

import (
	"fmt"
	"log"

	"github.com/horsley/svrkit"
)

func main() {
	svrkit.SwitchPwd()

	R.LoadStationNames("station_name.js")

	date := "2019-02-15"
	fmt.Println(date)

	train1 := commonGetLines(date, stationShenZhenBei, stationGuangZhouNan)
	train2 := commonGetLines(date, stationGuangZhouNan, stationRongGui)

	fmt.Println("去程:")
	transfer := resolveTransfer(train1, train2)
	for _, l := range transfer {
		fmt.Println(l)
	}

	train3 := commonGetLines(date, stationNanTou, stationGuangZhouNan)
	train4 := commonGetLines(date, stationGuangZhouNan, stationShenZhenBei)

	fmt.Println("返程:")
	transfer = resolveTransfer(train3, train4)
	for _, l := range transfer {
		fmt.Println(l)
	}
}

const (
	//广州南换乘间隔
	minTransferDuration = 25
	maxTransferDuration = 60
)

//计算换乘 要求传入的线路数组时间有序 返回换乘的成对信息
func resolveTransfer(firstLines, secondLines []lineInfo) []trainTransfer {
	result := make([]trainTransfer, 0)

	for i := 0; i < len(firstLines); i++ {
		firstTrainArrive := firstLines[i][lineInfoArriveTime]
		for j := 0; j < len(secondLines); j++ {
			gap := R.TimeGap(secondLines[j][lineInfoStartTime], firstTrainArrive)
			if gap < minTransferDuration || gap > maxTransferDuration {
				continue
			}

			result = append(result, trainTransfer{
				firstLines[i], secondLines[j],
			})
		}
	}
	return result
}

func commonGetLines(date, from, to string) []lineInfo {
	lines, err := R.QueryTickets(date, from, to)
	if err != nil {
		log.Panicln(err)
	}
	lines = R.FilterLinesCanNotBuy(lines)
	lines = R.FilterStationNotMatch(lines, from, to)
	lines = R.FilterLinesWithoutSeatGrade2(lines)

	return lines
}

func getLinesNanTou2GuangZhou(date string) []lineInfo {
	lines, err := R.QueryTickets(date, stationNanTou, stationGuangZhouNan)
	if err != nil {
		log.Panicln(err)
	}
	lines = R.FilterLinesCanNotBuy(lines)
	lines = R.FilterStationNotMatch(lines, stationNanTou, stationGuangZhouNan)
	lines = R.FilterLinesWithoutSeatGrade2(lines)

	return lines
}
