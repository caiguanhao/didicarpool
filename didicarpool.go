package didicarpool

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

type (
	Client struct {
		Token string
	}

	Orders struct {
		Orders    []Order
		NextMonth string
	}

	Location struct {
		Name string
		Lat  string
		Lon  string
	}

	Order struct {
		Id              string
		Exclusive       bool
		TotalPassengers int
		TotalAmount     string
		Routes          []Route
	}

	Route struct {
		Id         string
		UserId     string
		CreatedAt  time.Time
		StartedAt  time.Time
		From       Location
		To         Location
		Passengers int
		Amount     string
	}

	originalOrder struct {
		ID                  string `json:"id"`
		OrderID             string `json:"order_id"`
		Status              string `json:"status"`
		StatusDescColor     string `json:"status_desc_color"`
		OrderStateDesc      string `json:"order_state_desc"`
		StatusTxt           string `json:"status_txt"`
		Substatus           string `json:"substatus"`
		OrderNewStatus      string `json:"order_new_status"`
		DepartureTime       string `json:"departure_time"`
		DepartureTimeStatus string `json:"departure_time_status"`
		PayPrice            string `json:"pay_price"`
		Type                string `json:"type"`
		StatusDesc          string `json:"status_desc"`
		BusinessArea        string `json:"business_area"`
		SessionID           string `json:"session_id"`
		IsCarpool           string `json:"is_carpool"`
		UserInfo            struct {
			CarAuthStatus string `json:"car_auth_status"`
			UID           string `json:"uid"`
			HeadImgURL    string `json:"head_img_url"`
			NickName      string `json:"nick_name"`
			AuthStatus    string `json:"auth_status"`
		} `json:"user_info"`
		RouteInfo struct {
			FromAreaID     string `json:"from_area_id"`
			ToAreaID       string `json:"to_area_id"`
			FromLng        string `json:"from_lng"`
			FromLat        string `json:"from_lat"`
			FromName       string `json:"from_name"`
			FromAddress    string `json:"from_address"`
			ToLng          string `json:"to_lng"`
			ToLat          string `json:"to_lat"`
			ToName         string `json:"to_name"`
			ToAddress      string `json:"to_address"`
			RouteID        string `json:"route_id"`
			CountryIsoCode string `json:"country_iso_code"`
		} `json:"route_info"`
		IscanDelete         string `json:"iscan_delete"`
		CarpoolID           string `json:"carpool_id"`
		OrderStatus         string `json:"order_status"`
		StriveTime          string `json:"strive_time"`
		DepartureDistance   string `json:"departure_distance"`
		ToDepartureDistance string `json:"to_departure_distance"`
		JumpScheme          string `json:"jump_scheme"`
		TripDesc            []struct {
			Message string `json:"message"`
			Color   string `json:"color"`
		} `json:"trip_desc"`
		OrderGroup []originalOrder `json:"order_group"`
	}

	originalOrders struct {
		Delinfo struct {
			Status string `json:"status"`
			Msg    string `json:"msg"`
		} `json:"delinfo"`
		Orders    []originalOrder `json:"orders"`
		Next      string          `json:"next"`
		NextMonth string          `json:"next_month"`
		NextText  struct {
			Message string `json:"message"`
		} `json:"next_text"`
		Errno     string `json:"errno"`
		Errmsg    string `json:"errmsg"`
		Requestid string `json:"requestid"`
		Traceid   string `json:"traceid"`
	}
)

// Get list of orders in month. To get orders of current month, set month to
// empty. For other month, use yyyyMM format.
func (client Client) GetOrders(ctx context.Context, month string) (*Orders, error) {
	v := url.Values{}
	v.Set("token", client.Token)
	v.Set("role", "driver")
	v.Set("month", month)
	v.Set("limit", "60")
	url := url.URL{
		Scheme:   "https",
		Host:     "api.didialift.com",
		Path:     "beatles/orderapi/base/user/getorderlistv2",
		RawQuery: v.Encode(),
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	var orders originalOrders
	err = json.NewDecoder(res.Body).Decode(&orders)
	if err != nil {
		return nil, err
	}
	o := orders.toOrders()
	return &o, nil
}

func (o originalOrders) toOrders() Orders {
	var orders []Order
	for _, order := range o.Orders {
		var routes []Route
		for _, o := range order.OrderGroup {
			if o.Status == "4" {
				routes = append(routes, o.toRoute())
			}
		}
		if len(routes) == 0 {
			if order.Status == "4" {
				routes = append(routes, order.toRoute())
			}
		}
		if len(routes) == 0 {
			continue
		}
		var totalPassengers int
		for _, r := range routes {
			totalPassengers += r.Passengers
		}
		orders = append(orders, Order{
			Id:              order.OrderID,
			Exclusive:       order.IsCarpool == "0",
			TotalPassengers: totalPassengers,
			TotalAmount:     order.PayPrice,
			Routes:          routes,
		})
	}
	return Orders{
		Orders:    orders,
		NextMonth: o.NextMonth,
	}
}

func (order originalOrder) toRoute() Route {
	createdAt, startedAt := order.getTimes()
	return Route{
		Id:        order.OrderID,
		UserId:    order.UserInfo.UID,
		CreatedAt: createdAt,
		StartedAt: startedAt,
		From: Location{
			Name: order.RouteInfo.FromName,
			Lat:  order.RouteInfo.FromLat,
			Lon:  order.RouteInfo.FromLng,
		},
		To: Location{
			Name: order.RouteInfo.ToName,
			Lat:  order.RouteInfo.ToLat,
			Lon:  order.RouteInfo.ToLng,
		},
		Passengers: order.getPassengers(),
		Amount:     order.PayPrice,
	}
}

var (
	rePassengers         = regexp.MustCompile(`(\d)人`)
	reDepartureTimeMonth = regexp.MustCompile(`\d+月\d+日`)
	reDepartureTimeTime  = regexp.MustCompile(`\d+:\d+`)
	timezone             = time.FixedZone("UTC+8", 8*60*60)
)

func (order originalOrder) getPassengers() int {
	var passengers int
	for _, desc := range order.TripDesc {
		match := rePassengers.FindStringSubmatch(desc.Message)
		if match != nil {
			p, _ := strconv.Atoi(match[1])
			passengers += p
		}
	}
	return passengers
}

func (order originalOrder) getTimes() (createdAt, startedAt time.Time) {
	createdAt, _ = time.ParseInLocation("2006-01-02 15:04:05", order.StriveTime, timezone)
	month, _ := time.ParseInLocation("01月02日", reDepartureTimeMonth.FindString(order.DepartureTime), timezone)
	t, _ := time.ParseInLocation("15:04", reDepartureTimeTime.FindString(order.DepartureTime), timezone)
	startedAt = time.Date(createdAt.Year(), month.Month(), month.Day(), t.Hour(), t.Minute(), 0, 0, timezone)
	if startedAt.Before(createdAt) {
		startedAt = time.Date(createdAt.Year()+1, month.Month(), month.Day(), t.Hour(), t.Minute(), 0, 0, timezone)
	}
	return
}
