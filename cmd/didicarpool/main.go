package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/caiguanhao/didicarpool"
)

func main() {
	format := flag.String("f", "csv", "output format")
	number := flag.Int("n", 0, "number of months to get")
	output := flag.String("o", "", "write output to file name, to stdout if empty")
	flag.Parse()

	client := didicarpool.Client{
		Token: os.Getenv("DIDITOKEN"),
	}
	ctx := context.Background()
	var month string
	var count int
	var emptyMonth int
	var orders []didicarpool.Order
	for emptyMonth < 3 && (*number == 0 || count < *number) {
		if month == "" {
			log.Println("Getting data of current month")
		} else {
			log.Println("Getting data of month", month)
		}
		o, err := client.GetOrders(ctx, month)
		if err != nil {
			log.Fatal(err)
		}
		orders = append(orders, o.Orders...)
		month = o.NextMonth
		if len(o.Orders) == 0 {
			emptyMonth += 1
		} else {
			emptyMonth = 0
		}
		count += 1
	}
	var o io.Writer
	if *output == "" {
		o = os.Stdout
	} else {
		f, err := os.OpenFile(*output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			log.Fatal(err)
		}
		o = f
		defer f.Close()
	}
	switch *format {
	case "json":
		e := json.NewEncoder(o)
		e.SetIndent("", "  ")
		e.Encode(orders)
	default:
		fmt.Fprintln(o, "ID,接单时间,出发时间,独享,乘客数,价格")
		for _, order := range orders {
			var excl string
			if order.Exclusive {
				excl = "是"
			} else {
				excl = "否"
			}
			createdAt, startedAt := getTimes(order)
			fmt.Fprintf(o, "%s,%s,%s,%s,%d,%s\n",
				order.Id, formatTime(createdAt), formatTime(startedAt),
				excl, order.TotalPassengers, order.TotalAmount)
		}
	}
}

func getTimes(order didicarpool.Order) (createdAt, startedAt *time.Time) {
	for _, route := range order.Routes {
		if createdAt == nil || createdAt.After(route.CreatedAt) {
			createdAt = &route.CreatedAt
		}
		if startedAt == nil || startedAt.After(route.StartedAt) {
			startedAt = &route.StartedAt
		}
	}
	return
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02 15:04")
}
