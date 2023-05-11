package main

import (
    "github.com/valyala/fasthttp"
    "crypto/tls"
    "io/ioutil"
    "net/http"
    "bytes"
    "time"
)

func createFastHttpClient() *fasthttp.Client {
    return &fasthttp.Client{} // We don't use fasthttp for sniping so don't need to make the client anything special
}

func createNetHttpClient() *http.Client {
    return &http.Client{
        Transport: &http.Transport{
            TLSClientConfig:     &tls.Config{InsecureSkipVerify: true,},
            DisableKeepAlives:   false,
            MaxIdleConnsPerHost: 1000,
            ForceAttemptHTTP2:   true,
            DisableCompression:  true,
            IdleConnTimeout:     0,
            MaxIdleConns:        0,
            MaxConnsPerHost:     0,
        },
        Timeout:                 0,
    }
}

func buildClaimHeaders() {
    claimRequestHeaders = http.Header{
        "Content-Type": {"application/json"},
        "Authorization": {claimToken},
        "User-Agent": {userAgent},
    }
}

func snipeNitro(giftId string, start time.Time) (int, string, time.Duration) {
    var request, requestErr = http.NewRequest("POST", "https://discord.com/api/v" + apiVersion + "/entitlements/gift-codes/" + giftId + "/redeem", bytes.NewBuffer([]byte(`{}`)))

    if requestErr != nil { return 0, requestErr.Error(), time.Now().Sub(start) }

    request.Header = claimRequestHeaders

    var response, responseErr = discordClient.Do(request)

    if responseErr != nil { return 0, responseErr.Error(), time.Now().Sub(start) }

    defer response.Body.Close(); bodyBytes, _ := ioutil.ReadAll(response.Body)
    
    return response.StatusCode, string(bodyBytes), time.Now().Sub(start)

    // Update how 2 things are handled in this function and it'll improve speeds by a lot (figure it out xo)
}

func checkRateLimit() (bool, string) {
    var request, requestErr = http.NewRequest("GET", "https://discord.com/api/v" + apiVersion + "/invites/xo", nil)

    if requestErr != nil { return true, "Unknown" }

    var response, responseErr = discordClient.Do(request)

    if responseErr != nil { return true, "Unknown" }

    defer response.Body.Close();

    if response.StatusCode == 429 {
        return true, response.Header.Get("retry-after");
    } else { 
        return false, ""
    }
}

func discordPost(url string, data string) { var request *fasthttp.Request = fasthttp.AcquireRequest(); defer fasthttp.ReleaseRequest(request); request.Header.SetMethod("POST"); request.SetRequestURI(url); request.Header.Set("User-Agent", "Tsukuyomi/XO"); request.Header.Set("Content-Type", "application/json"); request.SetBody([]byte(data)); webhookClient.Do(request, nil); }
