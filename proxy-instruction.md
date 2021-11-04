To use the proxy:

1. Start the main server `go run cmd/server/main.go`
2. Start the proxy server `go run cmd/proxy/main.go -open_order_amount_limit=50 -open_order_sum_limit=5000`
3. Start clients `go run cmd/client/main.go -inst=USDCND -inter=0.03s`
   , `go run cmd/client/main.go -inst=RUBKZT -inter=0.05s`