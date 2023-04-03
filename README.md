# didicarpool

Get didi's car pool orders (滴滴顺风车订单).

To obtain `DIDITOKEN`, launch `mitmproxy`, set up phone's proxy and certificate
settings, launch the didi app on your phone, open the car pool home page, find
request whose URL starts with `https://api.didialift.com`, select that request,
and get the value of the cookie "ntoken".
