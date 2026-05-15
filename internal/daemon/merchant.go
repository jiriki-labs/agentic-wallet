package daemon

import "encoding/json"

// parseMerchantOrderID extracts orderId from a merchant JSON body (e.g. Grocery402 POST /orders).
func parseMerchantOrderID(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var out struct {
		OrderID string `json:"orderId"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return ""
	}
	return out.OrderID
}
