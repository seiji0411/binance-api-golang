package binance_api

import (
	"fmt"
	binance2 "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/davecgh/go-spew/spew"
	"github.com/kjeih/go-binance-api"
	"strconv"
	"sync"
	"testing"
	"time"
)

func subscribePrice() {
	recv := make(chan futures.WsDepthEvent)
	go binance_api.SubscribeFuturesOrderbook("SOLBUSD", recv)
	for {
		priceData := <-recv
		bidPrice, _, err := priceData.Bids[0].Parse()
		if err != nil {
			spew.Dump("subscribePrice Error" + err.Error())
			continue
		}
		askPrice, _, err := priceData.Asks[0].Parse()
		spew.Dump(fmt.Sprintf("OrderBook Now %d => event: %d; bid: %.4f, ask: %.4f", time.Now().UnixMilli(), priceData.Time, bidPrice, askPrice))
	}
}

func subscribeKPrice() {
	recv := make(chan futures.WsKlineEvent)
	go binance_api.SubscribeFuturesKline("SOLBUSD", recv)
	for {
		priceData := <-recv
		openPrice, err := strconv.ParseFloat(priceData.Kline.Open, 64)
		if err != nil {
			spew.Dump("subscribeKPrice Error" + err.Error())
			continue
		}
		closePrice, err := strconv.ParseFloat(priceData.Kline.Close, 64)
		if err != nil {
			spew.Dump("subscribeKPrice Error" + err.Error())
			continue
		}
		spew.Dump(fmt.Sprintf("Kline Now %d => event: %d; open: %.4f, close: %.4f", time.Now().UnixMilli(), priceData.Time, openPrice, closePrice))
	}
}

func subscribeSpotPrice(symbol string) {
	recv := make(chan *binance2.WsPartialDepthEvent)
	go binance_api.SubscribeSpotPrice(symbol, recv)
	for {
		priceData := <-recv
		bidPrice, _, err := priceData.Bids[0].Parse()
		if err != nil {
			spew.Dump("subscribeSpotPrice Error" + err.Error())
			continue
		}
		askPrice, _, err := priceData.Asks[0].Parse()
		spew.Dump(fmt.Sprintf("%s Now %d => bid: %.5f, ask: %.5f", symbol, time.Now().UnixMilli(), bidPrice, askPrice))
	}
}

func subscribeSpotKlinePrice(symbol string) {
	recv := make(chan *binance2.WsKlineEvent)
	go binance_api.SubscribeSpotKlinePrice(symbol, recv)
	for {
		priceData := <-recv
		openPrice, err := strconv.ParseFloat(priceData.Kline.Open, 64)
		if err != nil {
			spew.Dump("subscribeKPrice Error" + err.Error())
			continue
		}
		closePrice, err := strconv.ParseFloat(priceData.Kline.Close, 64)
		if err != nil {
			spew.Dump("subscribeKPrice Error" + err.Error())
			continue
		}
		spew.Dump(fmt.Sprintf("%s Kline Now %d => event: %d; open: %.4f, close: %.4f", symbol, time.Now().UnixMilli(), priceData.Time, openPrice, closePrice))
	}
}

func getPosition(apiKey, secretKey string) {
	service := binance_api.NewBinanceService(apiKey, secretKey)
	pos, err := service.GetPosition("SOLBUSD")
	if err != nil {
		print(err)
	} else {
		fmt.Printf("Pos: %.3f\n", pos)
	}
}

func buy(apiKey, secretKey string) {
	service := binance_api.NewBinanceService(apiKey, secretKey)
	result, err := service.Buy("SOLBUSD", 1)
	if err != nil {
		print(err)
	} else {
		spew.Dump(result)
	}
}

func sell(apiKey, secretKey string) {
	service := binance_api.NewBinanceService(apiKey, secretKey)
	result, err := service.Sell("SOLBUSD", 1)
	if err != nil {
		print(err)
	} else {
		spew.Dump(result)
	}
}

func TestPriceSubscription(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go subscribeSpotPrice("USDCUSDT")
	go subscribeSpotKlinePrice("USDCUSDT")
	go subscribeSpotPrice("BUSDUSDT")
	go subscribeSpotKlinePrice("BUSDUSDT")
	wg.Wait()

	time.Sleep(3 * time.Second)
}

var BINANCE_API_KEY = "apiKey"
var BINANCE_SECRET_KEY = "secretKey"

func TestPosition(t *testing.T) {
	getPosition(BINANCE_API_KEY, BINANCE_SECRET_KEY)

	buy(BINANCE_API_KEY, BINANCE_SECRET_KEY)
	time.Sleep(3 * time.Second)

	sell(BINANCE_API_KEY, BINANCE_SECRET_KEY)
	time.Sleep(3 * time.Second)

	getPosition(BINANCE_API_KEY, BINANCE_SECRET_KEY)
}
