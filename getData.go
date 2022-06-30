package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
)

type KLineData struct {
	highs      []float64
	lows       []float64
	closes     []float64
	opens      []float64
	volumes    []float64
	takers     []float64
	makers     []float64
	times      []string
	opentimes  []time.Time
	closetimes []time.Time
}

func str2float64(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return f
}

func GetSymbolTickSize(symbol string) (tickSize float64) {
	apiKey := os.Getenv("API_KEY")
	secretKey := os.Getenv("SECRET_KEY")
	client := binance.NewClient(apiKey, secretKey)
	symbolInfo, err := client.NewExchangeInfoService().Symbol(symbol).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	return str2float64(symbolInfo.Symbols[0].Filters[0]["tickSize"].(string))
}

func GetKLineData(symbol string, interval string) (ret KLineData) {
	apiKey := os.Getenv("API_KEY")
	secretKey := os.Getenv("SECRET_KEY")
	client := binance.NewClient(apiKey, secretKey)
	klines, err := client.NewKlinesService().Symbol(symbol).
		Interval(interval).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	var highs, lows, closes, opens, volumes, takers, makers []float64
	var times []string
	var opentimes, closetimes []time.Time
	for _, k := range klines {
		highs = append(highs, str2float64(k.High))
		lows = append(lows, str2float64(k.Low))
		closes = append(closes, str2float64(k.Close))
		opens = append(opens, str2float64(k.Open))
		v := str2float64(k.Volume)
		t := str2float64(k.TakerBuyBaseAssetVolume)
		volumes = append(volumes, v)
		takers = append(takers, t)
		makers = append(makers, v-t)
		opentime := time.Unix(k.OpenTime/1000.0, 0)
		closetime := time.Unix(k.CloseTime/1000.0, 0)
		opentimes = append(opentimes, opentime)
		closetimes = append(closetimes, closetime)
		times = append(times, fmt.Sprintf("%d/%02d/%02d %02d:%02d",
			opentime.Year(),
			opentime.Month(),
			opentime.Day(),
			opentime.Hour(),
			opentime.Minute(),
		))
	}
	var kline KLineData
	kline.highs = highs
	kline.lows = lows
	kline.closes = closes
	kline.opens = opens
	kline.makers = makers
	kline.takers = takers
	kline.times = times
	kline.volumes = volumes
	kline.opentimes = opentimes
	kline.closetimes = closetimes
	return kline
}
