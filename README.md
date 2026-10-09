# Binance-API-Golang

A small Go wrapper around [`adshao/go-binance`](https://github.com/adshao/go-binance) for **Binance Spot and USDⓈ-M Futures**. It gives you:

- **Real-time market data** over WebSocket, delivered straight into Go channels (order book depth and klines, spot and futures)
- **Futures trading helpers** for one-call market `Buy` / `Sell` orders
- **Position lookup** for any futures symbol
- **No config files.** You pass your API keys to the constructor and load them however you like.

![Go](https://img.shields.io/badge/Go-1.20%2B-00ADD8?logo=go&logoColor=white)
![Binance](https://img.shields.io/badge/Binance-Spot%20%7C%20Futures-F0B90B?logo=binance&logoColor=black)

> [!WARNING]
> `Buy`, `Sell` and `NewOrder` place **real market orders** on your futures account, and so does the `TestPosition` test. Read [Safety notes](#safety-notes) and use the [testnet](#using-the-binance-testnet) while you develop.

---

## Table of contents

- [Requirements](#requirements)
- [Installation](#installation)
- [API keys](#api-keys)
- [Quick start](#quick-start)
- [Usage](#usage)
  - [Market data streams](#market-data-streams)
  - [Futures trading](#futures-trading)
  - [Positions](#positions)
  - [User data stream](#user-data-stream)
  - [Using the Binance testnet](#using-the-binance-testnet)
  - [Accessing the underlying clients](#accessing-the-underlying-clients)
- [API reference](#api-reference)
- [Testing](#testing)
- [Project layout](#project-layout)
- [Safety notes](#safety-notes)
- [Troubleshooting](#troubleshooting)

---

## Requirements

| Requirement     | Notes                                                                   |
|-----------------|-------------------------------------------------------------------------|
| Go **1.20+**    | As declared in `go.mod`                                                 |
| Binance account | With an API key. Futures trading must be enabled to place orders.       |
| Network access  | To `api.binance.com`, `fapi.binance.com` and their WebSocket endpoints  |

Public market data streams need **no API key at all**. Keys are only needed for trading, positions and user data.

## Installation

Clone the repo and download dependencies:

```bash
git clone https://github.com/seiji0411/binance-api-golang.git
cd binance-api-golang
go mod download
go build ./...
```

### Using it from another module

The module path declared in `go.mod` is `github.com/kjeih/go-binance-api`, and the package name is `binance_api`:

```go
import binance_api "github.com/kjeih/go-binance-api"
```

The module path is not the same as this repository's URL. If you consume a local clone, point the module path at it with a `replace` directive in **your** `go.mod`:

```go
require github.com/kjeih/go-binance-api v0.0.0

replace github.com/kjeih/go-binance-api => ../binance-api-golang
```

## API keys

The library doesn't read config files or environment variables itself. You pass your keys directly to `NewBinanceService`:

```go
svc := binance_api.NewBinanceService(apiKey, secretKey)
```

The recommended way to load them is from environment variables:

```go
apiKey := os.Getenv("BINANCE_API_KEY")
secretKey := os.Getenv("BINANCE_SECRET_KEY")
if apiKey == "" || secretKey == "" {
	log.Fatal("BINANCE_API_KEY and BINANCE_SECRET_KEY must be set")
}
svc := binance_api.NewBinanceService(apiKey, secretKey)
```

```bash
# bash / zsh
export BINANCE_API_KEY="your-api-key"
export BINANCE_SECRET_KEY="your-secret-key"
```

```powershell
# PowerShell
$env:BINANCE_API_KEY    = "your-api-key"
$env:BINANCE_SECRET_KEY = "your-secret-key"
```

> [!CAUTION]
> Never hard-code real keys in source files that you commit, including test files.

## Quick start

A minimal program that streams the best bid and ask for `BTCUSDT`. No keys are needed:

```go
package main

import (
	"fmt"

	"github.com/adshao/go-binance/v2"
	binance_api "github.com/kjeih/go-binance-api"
)

func main() {
	recv := make(chan *binance.WsPartialDepthEvent)
	go binance_api.SubscribeSpotPrice("BTCUSDT", recv)

	for ev := range recv {
		bid, _, _ := ev.Bids[0].Parse()
		ask, _, _ := ev.Asks[0].Parse()
		fmt.Printf("BTCUSDT  bid=%.2f  ask=%.2f  spread=%.2f\n", bid, ask, ask-bid)
	}
}
```

```bash
go run .
```

---

## Usage

### Market data streams

Every `Subscribe*` function:

1. opens a WebSocket for `symbol`,
2. pushes each event into the channel you pass in, and
3. **blocks until the connection closes**. Always start it with `go`.

| Function                    | Market  | Stream                  | Channel type                        | Rate / interval |
|-----------------------------|---------|-------------------------|-------------------------------------|-----------------|
| `SubscribeSpotPrice`        | Spot    | Partial depth, 5 levels | `chan *binance.WsPartialDepthEvent` | 100 ms          |
| `SubscribeSpotKlinePrice`   | Spot    | Kline                   | `chan *binance.WsKlineEvent`        | `1s` candles    |
| `SubscribeFuturesOrderbook` | Futures | Partial depth, 5 levels | `chan futures.WsDepthEvent`         | 100 ms          |
| `SubscribeFuturesKline`     | Futures | Kline                   | `chan futures.WsKlineEvent`         | `1m` candles    |

> [!NOTE]
> The handlers write to your channel synchronously. If you use an unbuffered channel and stop reading, the WebSocket read loop stalls. Use a buffered channel (`make(chan T, 100)`) if your consumer can be slow.

#### Spot order book (top 5 levels)

```go
recv := make(chan *binance.WsPartialDepthEvent, 100)
go binance_api.SubscribeSpotPrice("ETHUSDT", recv)

for ev := range recv {
	bid, bidQty, _ := ev.Bids[0].Parse()
	ask, askQty, _ := ev.Asks[0].Parse()
	fmt.Printf("bid %.2f x %.4f | ask %.2f x %.4f\n", bid, bidQty, ask, askQty)
}
```

#### Spot klines (1-second candles)

```go
recv := make(chan *binance.WsKlineEvent, 100)
go binance_api.SubscribeSpotKlinePrice("ETHUSDT", recv)

for ev := range recv {
	k := ev.Kline
	fmt.Printf("[%d] O=%s H=%s L=%s C=%s V=%s closed=%v\n",
		k.StartTime, k.Open, k.High, k.Low, k.Close, k.Volume, k.IsFinal)
}
```

#### Futures order book

```go
recv := make(chan futures.WsDepthEvent, 100)
go binance_api.SubscribeFuturesOrderbook("SOLUSDT", recv)

for ev := range recv {
	bid, _, _ := ev.Bids[0].Parse()
	ask, _, _ := ev.Asks[0].Parse()
	fmt.Printf("event=%d bid=%.4f ask=%.4f\n", ev.Time, bid, ask)
}
```

#### Futures klines (1-minute candles)

```go
recv := make(chan futures.WsKlineEvent, 100)
go binance_api.SubscribeFuturesKline("SOLUSDT", recv)

for ev := range recv {
	open, _ := strconv.ParseFloat(ev.Kline.Open, 64)
	closeP, _ := strconv.ParseFloat(ev.Kline.Close, 64)
	fmt.Printf("open=%.4f close=%.4f change=%.2f%%\n", open, closeP, (closeP-open)/open*100)
}
```

#### Several streams at once

```go
spot := make(chan *binance.WsPartialDepthEvent, 100)
fut := make(chan futures.WsDepthEvent, 100)

go binance_api.SubscribeSpotPrice("SOLUSDT", spot)
go binance_api.SubscribeFuturesOrderbook("SOLUSDT", fut)

var spotMid, futMid float64
for {
	select {
	case ev := <-spot:
		b, _, _ := ev.Bids[0].Parse()
		a, _, _ := ev.Asks[0].Parse()
		spotMid = (a + b) / 2
	case ev := <-fut:
		b, _, _ := ev.Bids[0].Parse()
		a, _, _ := ev.Asks[0].Parse()
		futMid = (a + b) / 2
	}
	if spotMid > 0 && futMid > 0 {
		fmt.Printf("basis: %.4f (%.3f%%)\n", futMid-spotMid, (futMid-spotMid)/spotMid*100)
	}
}
```

> [!WARNING]
> Stream errors **panic**. A connect error panics in your goroutine, so you can recover from it. A mid-stream error (such as a dropped connection) panics inside go-binance's own goroutine, where you **cannot** recover from it, and the process exits. For long-running bots that must survive disconnects, call `binance.WsPartialDepthServe100Ms`, `futures.WsKlineServe` and the other go-binance functions directly with your own error handler. See [Troubleshooting](#troubleshooting).

### Futures trading

Create a service with your keys. It builds both a spot and a futures REST client:

```go
svc := binance_api.NewBinanceService(apiKey, secretKey)
```

#### Market buy / sell

```go
order, err := svc.Buy(binance_api.SymbolSOLUSDT, 1) // market BUY 1 SOL
if err != nil {
	log.Fatal(err)
}
fmt.Printf("order %d filled %s @ avg %s\n", order.OrderID, order.ExecutedQuantity, order.AvgPrice)

order, err = svc.Sell(binance_api.SymbolSOLUSDT, 1) // market SELL 1 SOL
```

`Buy` and `Sell` take an **integer** quantity. For fractional sizes, call `NewOrder` directly with a string quantity:

```go
import "github.com/adshao/go-binance/v2/futures"

order, err := svc.NewOrder("BTCUSDT", futures.SideTypeBuy, "0.003")
```

Every order is sent as:

| Parameter      | Value                                              |
|----------------|----------------------------------------------------|
| `type`         | `MARKET`                                           |
| `positionSide` | `BOTH` (requires **One-way Mode** on your account) |
| `reduceOnly`   | `false`                                            |

Errors are returned to you and are not logged by the library.

#### `FutureOrder` fields that are populated

`NewOrder` copies these fields from the exchange response: `Symbol`, `OrderID`, `ClientOrderID`, `Price`, `AvgPrice`, `ExecutedQuantity`, `CumQuote`, `ActivatePrice`. The other fields of the struct stay at their zero value.

### Positions

```go
amt, err := svc.GetPosition("SOLUSDT")
if err != nil {
	log.Fatal(err)
}

switch {
case amt > 0:
	fmt.Printf("LONG %.3f\n", amt)
case amt < 0:
	fmt.Printf("SHORT %.3f\n", -amt)
default:
	fmt.Println("FLAT")
}
```

`GetPosition` returns the signed position amount (`positionAmt`): positive for long, negative for short, `0` when flat. It reads the first entry returned by Binance, which is the correct one in One-way Mode.

### User data stream

```go
svc.SubscribeFutureUserData()
```

This only **starts** a futures user-data stream and prints the listen key. It does not connect to it yet. To consume events, pass the key to `futures.WsUserDataServe` from `go-binance`, and keep it alive with `svc.FutureClient.NewKeepaliveUserStreamService()` every 30–60 minutes.

### Using the Binance testnet

`go-binance` has global testnet switches. Set them **before** `NewBinanceService()` or any `Subscribe*` call, and use keys from [testnet.binancefuture.com](https://testnet.binancefuture.com) (futures) or [testnet.binance.vision](https://testnet.binance.vision) (spot):

```go
import (
	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
)

func main() {
	binance.UseTestnet = true
	futures.UseTestnet = true

	svc := binance_api.NewBinanceService(os.Getenv("BINANCE_API_KEY"), os.Getenv("BINANCE_SECRET_KEY"))
	// orders now go to the testnet
}
```

### Accessing the underlying clients

`BinanceService` exposes the raw `go-binance` clients, so you can use any endpoint this wrapper does not cover:

```go
// Spot account balances
acct, _ := svc.Client.NewGetAccountService().Do(context.Background())
for _, b := range acct.Balances {
	if b.Free != "0.00000000" {
		fmt.Println(b.Asset, b.Free)
	}
}

// Futures: set leverage
svc.FutureClient.NewChangeLeverageService().Symbol("SOLUSDT").Leverage(5).Do(context.Background())

// Futures: limit order
svc.FutureClient.NewCreateOrderService().
	Symbol("SOLUSDT").Side(futures.SideTypeBuy).
	Type(futures.OrderTypeLimit).TimeInForce(futures.TimeInForceTypeGTC).
	Quantity("1").Price("120.00").
	Do(context.Background())
```

---

## API reference

### Service — `binance.go`

| Symbol                                                 | Returns                 | Description                                         |
|--------------------------------------------------------|-------------------------|-----------------------------------------------------|
| `NewBinanceService(apiKey, secretKey string)`          | `*BinanceService`       | Builds spot and futures clients from your keys.     |
| `(*BinanceService).Buy(symbol, amount int)`            | `(*FutureOrder, error)` | Futures market buy.                                 |
| `(*BinanceService).Sell(symbol, amount int)`           | `(*FutureOrder, error)` | Futures market sell.                                |
| `(*BinanceService).NewOrder(symbol, side, qty string)` | `(*FutureOrder, error)` | Futures market order with any side and quantity.    |
| `(*BinanceService).GetPosition(symbol)`                | `(float64, error)`      | Signed position amount.                             |
| `(*BinanceService).SubscribeFutureUserData()`          | —                       | Starts a user stream and prints the listen key.     |

`BinanceService` fields: `Client *binance.Client`, `FutureClient *futures.Client`, plus `Positions` and `Balances` (`map[string]int64`), which are reserved for your own bookkeeping and are not populated by the library.

### Streams — `binance.go`

| Function                                  | Blocks | Panics on error |
|-------------------------------------------|:------:|:---------------:|
| `SubscribeSpotPrice(symbol, chan)`        | ✅     | ✅              |
| `SubscribeSpotKlinePrice(symbol, chan)`   | ✅     | ✅              |
| `SubscribeFuturesOrderbook(symbol, chan)` | ✅     | ✅              |
| `SubscribeFuturesKline(symbol, chan)`     | ✅     | ✅              |

### Types — `types.go`

| Symbol           | Description                                        |
|------------------|----------------------------------------------------|
| `SymbolSOLBUSD`  | `"SOLBUSD"` (BUSD markets are delisted, see below) |
| `SymbolSOLUSDT`  | `"SOLUSDT"`                                        |
| `FutureOrder`    | Result of an order call.                           |
| `FuturePosition` | Position risk snapshot (used internally).          |

---

## Testing

The tests live in [`tests/binance_test.go`](tests/binance_test.go) and are **live integration tests**. They talk to the real Binance API, so you need network access. The `tests` package imports the library exactly like an external consumer would, so they also check the public API.

| Test                    | What it does                                                                       | Needs keys | Real money |
|-------------------------|------------------------------------------------------------------------------------|:----------:|:----------:|
| `TestPriceSubscription` | Streams spot depth and 1s klines for `USDCUSDT` and `BUSDUSDT` and prints every event | | |
| `TestPosition`          | Gets the position, market **buys 1** `SOLBUSD`, market **sells 1**, then gets the position again | ✅ | ⚠️ **yes** |

### Static checks

These run without network access or credentials:

```bash
go build ./...
go vet ./...
go test -run '^$' ./...   # compile all tests without running them
```

### Price subscription test

No keys are needed. **This test never returns by itself.** It waits on a `sync.WaitGroup` that is never released, so always set a timeout or stop it with `Ctrl+C`:

```bash
go test ./tests -run TestPriceSubscription -v -timeout 30s
```

When the timeout expires, Go reports `panic: test timed out after 30s`. That is expected here, because it is how the streaming test ends. Without `-timeout`, it runs for Go's default 10 minutes.

Sample output:

```
(string) (len=62) "USDCUSDT Now 1759915200123 => bid: 0.99990, ask: 1.00000"
(string) (len=95) "USDCUSDT Kline Now 1759915200130 => event: 1759915200101; open: 0.9999, close: 1.0000"
```

### Position / trading test

> [!CAUTION]
> `TestPosition` places **two real market orders**. Run it against the testnet.

1. Set the keys at the bottom of `tests/binance_test.go`. They are placeholders by default:

   ```go
   var BINANCE_API_KEY = "apiKey"
   var BINANCE_SECRET_KEY = "secretKey"
   ```

   To avoid committing real keys, read them from the environment instead:

   ```go
   var BINANCE_API_KEY = os.Getenv("BINANCE_API_KEY")
   var BINANCE_SECRET_KEY = os.Getenv("BINANCE_SECRET_KEY")
   ```

2. Switch the symbol: the helpers use `SOLBUSD`, which is **delisted**. Change it to `SOLUSDT`.

3. Turn on the testnet with an `init()` in the test file:

   ```go
   func init() {
   	binance2.UseTestnet = true // binance2 is the test file's alias for go-binance/v2
   	futures.UseTestnet = true
   }
   ```

4. Run it:

   ```bash
   go test ./tests -run TestPosition -v
   ```

   Expected output:

   ```
   Pos: 0.000
   (*binance_api.FutureOrder)(0xc000...)({ Symbol: "SOLUSDT", OrderID: 123456789, ... ExecutedQuantity: "1", ... })
   (*binance_api.FutureOrder)(0xc000...)({ Symbol: "SOLUSDT", ... })
   Pos: 0.000
   ```

> [!NOTE]
> The test helpers print errors with `print(err)` and do not call `t.Fatal`, so `TestPosition` **passes even when every call fails**. Read the output.

### Running everything

```bash
go test ./... -v -timeout 60s
```

This runs **both** tests, including the real-money `TestPosition`. Prefer `-run` to select one test.

### Writing your own tests

Stream functions only need a channel, so you can write tests that finish on their own with a bounded read instead of an endless loop:

```go
func TestSpotStreamDeliversEvents(t *testing.T) {
	recv := make(chan *binance2.WsPartialDepthEvent, 10)
	go binance_api.SubscribeSpotPrice("BTCUSDT", recv)

	select {
	case ev := <-recv:
		if len(ev.Bids) == 0 || len(ev.Asks) == 0 {
			t.Fatal("empty order book")
		}
		bid, _, _ := ev.Bids[0].Parse()
		ask, _, _ := ev.Asks[0].Parse()
		if bid >= ask {
			t.Fatalf("crossed book: bid %.2f >= ask %.2f", bid, ask)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no depth event within 10s")
	}
}
```

```bash
go test ./tests -run TestSpotStreamDeliversEvents -v
```

### Test logger

[`tests/logger.go`](tests/logger.go) contains an async, channel-backed logger (`Setup` / `Outlog`, `TargetSetup` / `TargetOutlog`, `WriteFile`) that is available inside the `tests` package. It writes RFC3339Nano UTC timestamped lines to stdout. The `filename` argument is currently ignored.

---

## Project layout

```
.
├── binance.go             # BinanceService, orders, positions, WebSocket subscriptions
├── types.go               # FutureOrder, FuturePosition, symbol constants
├── tests/
│   ├── binance_test.go    # Live integration tests
│   └── logger.go          # Async stdout logger used by tests
├── go.mod
└── go.sum
```

### Dependencies

| Module                                                        | Purpose                         |
|---------------------------------------------------------------|---------------------------------|
| [`adshao/go-binance/v2`](https://github.com/adshao/go-binance) | Binance REST + WebSocket client |
| [`davecgh/go-spew`](https://github.com/davecgh/go-spew)       | Pretty-printing in tests        |

---

## Safety notes

- **Real orders.** `Buy`, `Sell`, `NewOrder` and `TestPosition` send `MARKET` orders that fill immediately at whatever price is available. Develop on the [testnet](#using-the-binance-testnet).
- **Restrict your API key.** In the Binance API management page, enable only the permissions you need (for example, *Futures* without *Withdrawals*) and restrict it to your server's IP.
- **Keep secrets out of git.** Load keys from environment variables and don't hard-code them in source or test files.
- **One-way Mode.** Orders use `positionSide=BOTH`. In Hedge Mode, Binance rejects them.
- **Panics.** Mid-stream WebSocket errors in `Subscribe*` terminate the process.

## Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| `APIError(code=-2014): API-key format invalid` | The key is empty or still set to the `"apiKey"` placeholder. |
| `APIError(code=-2015): Invalid API-key, IP, or permissions` | Futures permission is not enabled, the IP is not on the key's whitelist, or you're using mainnet keys on the testnet (or the other way round). |
| `APIError(code=-4061): Order's position side does not match` | The account is in Hedge Mode. Switch to One-way Mode. |
| `APIError(code=-1121): Invalid symbol` | The symbol is delisted (for example, any `*BUSD` pair) or misspelled. |
| `APIError(code=-1111): Precision is over the maximum` | The quantity has too many decimals for that symbol. Check its `stepSize` via exchange info. |
| `panic: index out of range` in `GetPosition` | Binance returned no positions for the symbol, usually because the symbol is invalid on futures. |
| `TestPriceSubscription` never finishes | Expected. It streams forever. Use `-timeout`. |
| `panic: Binance Subscribe... error: websocket: close 1006` | The network dropped or Binance closed the connection (it closes connections after 24h). This panic can't be recovered. For automatic reconnects, use go-binance's `Ws*Serve` functions directly with an error handler that signals a reconnect instead of panicking. |
