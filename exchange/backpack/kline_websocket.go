package backpack

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"opensqt/logger"

	"github.com/gorilla/websocket"
)

type KlineWebSocketManager struct {
	mu           sync.RWMutex
	running      bool
	stopC        chan struct{}
	reconnectGap time.Duration
}

func NewKlineWebSocketManager() *KlineWebSocketManager {
	return &KlineWebSocketManager{reconnectGap: 5 * time.Second}
}

func (k *KlineWebSocketManager) Start(ctx context.Context, symbolMap map[string]string, interval string, callback func(candle interface{})) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.running {
		return fmt.Errorf("K线流已在运行")
	}

	k.running = true
	k.stopC = make(chan struct{})
	go k.connectLoop(ctx, k.stopC, symbolMap, interval, callback)
	return nil
}

func (k *KlineWebSocketManager) Stop() {
	k.mu.Lock()
	defer k.mu.Unlock()

	if !k.running {
		return
	}

	k.running = false
	close(k.stopC)
}

func (k *KlineWebSocketManager) connectLoop(ctx context.Context, stopC chan struct{}, symbolMap map[string]string, interval string, callback func(candle interface{})) {
	params := make([]string, 0, len(symbolMap))
	for marketSymbol := range symbolMap {
		params = append(params, fmt.Sprintf("kline.%s.%s", interval, marketSymbol))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-stopC:
			return
		default:
		}

		conn, _, err := websocket.DefaultDialer.Dial(backpackWSURL, nil)
		if err != nil {
			logger.Warn("⚠️ [Backpack] K线流连接失败: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-stopC:
				return
			case <-time.After(k.reconnectGap):
			}
			continue
		}

		subscribe := map[string]any{
			"method": "SUBSCRIBE",
			"params": params,
		}
		if err := conn.WriteJSON(subscribe); err != nil {
			conn.Close()
			logger.Warn("⚠️ [Backpack] K线订阅失败: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-stopC:
				return
			case <-time.After(k.reconnectGap):
			}
			continue
		}

		for {
			select {
			case <-ctx.Done():
				conn.Close()
				return
			case <-stopC:
				conn.Close()
				return
			default:
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				conn.Close()
				logger.Warn("⚠️ [Backpack] K线流断开，准备重连: %v", err)
				break
			}

			var payload struct {
				Stream string `json:"stream"`
				Data   struct {
					Symbol   string `json:"s"`
					Start    string `json:"t"`
					End      string `json:"T"`
					Open     string `json:"o"`
					Close    string `json:"c"`
					High     string `json:"h"`
					Low      string `json:"l"`
					Volume   string `json:"v"`
					IsClosed bool   `json:"X"`
				} `json:"data"`
			}
			if err := json.Unmarshal(message, &payload); err != nil {
				continue
			}

			startTime, err := parseBackpackTime(payload.Data.Start)
			if err != nil {
				continue
			}

			displaySymbol := symbolMap[payload.Data.Symbol]
			if displaySymbol == "" {
				displaySymbol = payload.Data.Symbol
			}

			callback(&Candle{
				Symbol:    displaySymbol,
				Open:      parseFloat(payload.Data.Open),
				High:      parseFloat(payload.Data.High),
				Low:       parseFloat(payload.Data.Low),
				Close:     parseFloat(payload.Data.Close),
				Volume:    parseFloat(payload.Data.Volume),
				Timestamp: startTime.UnixMilli(),
				IsClosed:  payload.Data.IsClosed,
			})
		}
	}
}
