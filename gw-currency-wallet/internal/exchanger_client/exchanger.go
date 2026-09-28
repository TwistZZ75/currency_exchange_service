package exchanger

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "proto_exchange/exchange"
)

type Client struct {
	conn *grpc.ClientConn
	api  pb.ExchangeServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", addr, err)
	}
	return &Client{conn: conn, api: pb.NewExchangeServiceClient(conn)}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) GetMap(ctx context.Context) (map[string]float64, error) {
	resp, err := c.api.GetExchangeMap(ctx, &pb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("grpc GetExchangeMap: %w", err)
	}
	out := make(map[string]float64, len(resp.GetRates()))
	for k, v := range resp.GetRates() {
		out[k] = float64(v)
	}
	return out, nil
}

// GetRate возвращает курс пары from -> to
func (c *Client) GetRate(ctx context.Context, from, to string) (float64, error) {
	resp, err := c.api.GetExchangeRateForCurrency(ctx, &pb.CurrencyRequest{
		FromCurrency: from,
		ToCurrency:   to,
	})
	if err != nil {
		return 0, fmt.Errorf("grpc GetExchangeRateForCurrency: %w", err)
	}
	return float64(resp.GetRate()), nil
}
