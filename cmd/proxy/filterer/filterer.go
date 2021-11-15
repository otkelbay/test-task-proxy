package filterer

import (
	"fmt"
	"sync"
	proxy "test.task/backend/proxy"
)

type Filterer struct {
	Limits               map[uint32]map[string][]float64
	OpenOrderAmountLimit int
	OpenOrderSumLimit    float64
	m                    sync.RWMutex
}

func New(openOrderSumLimit float64, openOrderAmountLimit int) *Filterer {
	filterer := Filterer{}
	filterer.m = sync.RWMutex{}
	filterer.OpenOrderAmountLimit = openOrderAmountLimit
	filterer.OpenOrderSumLimit = openOrderSumLimit
	filterer.Limits = make(map[uint32]map[string][]float64)

	return &filterer
}

func (f *Filterer) Filter(order *proxy.OrderRequest) bool {
	if f.OpenOrderAmountLimit == 0 {
		return false
	}
	if f.OpenOrderSumLimit == 0 {
		return false
	}

	f.m.RLock()
	_, ok := f.Limits[order.ClientID]
	f.m.RUnlock()
	if !ok {
		f.Limits[order.ClientID] = make(map[string][]float64)
	}

	f.m.RLock()
	_, ok = f.Limits[order.ClientID][order.Instrument]
	f.m.RUnlock()
	if !ok {
		if f.OpenOrderSumLimit < order.Volume {
			return false
		}
		f.Limits[order.ClientID][order.Instrument] = []float64{order.Volume}

		return true
	}

	f.m.Lock()
	clientLimit := f.Limits[order.ClientID][order.Instrument]
	fmt.Printf("client_id: %d sum: %f \n", order.ClientID, sum(clientLimit))
	fmt.Printf("client_id: %d amount: %d \n", order.ClientID, len(clientLimit))
	if len(clientLimit) >= f.OpenOrderAmountLimit {
		f.m.Unlock()
		return false
	}

	resSum := sum(clientLimit) + order.Volume
	if resSum > f.OpenOrderSumLimit {
		f.m.Unlock()
		return false
	}

	f.Limits[order.ClientID][order.Instrument] = append(f.Limits[order.ClientID][order.Instrument], order.Volume)
	f.m.Unlock()

	return true
}

func sum(arr []float64) float64 {
	var resSum float64
	resSum = 0
	for _, el := range arr {
		resSum += el
	}

	return resSum
}
