// go build -ldflags "-s -w" -o sniper *.go && strip -s sniper

// The websocket is buggy and will stop receiving data after X time
// I've removed everything related to my api so it's missing a few features like loading all the previous nitro's you've detected so it'll be per session now

package main

import (
	"sync/atomic"
	"math/rand"
	"os/signal"
	"strings"
    "runtime"
    "syscall"
    "time"
    "fmt"
	"os"
)

func main() {
	discordClient = createNetHttpClient(); apiClient = createFastHttpClient(); webhookClient = createFastHttpClient()
	rand.Seed(time.Now().UnixNano()); setULimit(); fmt.Printf(sniperAscii, sniperVersion); checkDataFolderExists(); config = loadConfig()

	claimToken = readSingleFile("./data/claimToken.txt")
	alts = readFile("./data/alts.txt")
	
	c := make(chan os.Signal); signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	if len(alts) == 0 { fmt.Printf("%s No alts detected, exiting...\n", ERROR); return }; if len(claimToken) < 5 { fmt.Printf("%s Your claim token isn't even 5 characters long, exiting...\n", ERROR); return }
	if config.Sniper.SnipeOnMain { alts = append(alts, claimToken) }; if config.Sniper.Threads > len(alts) { useAllThreads() }; if config.Sniper.Threads == 0 { useAllThreads() }
	if !config.Sniper.SaveInvites { fmt.Printf("%s I'd just like to inform you that you aren't saving the invites the Sniper detects\n", INFO) }

	sortAlts(); buildClaimHeaders()

	fmt.Printf("\n%s Successfully loaded \x1b[37m\x1b[38;5;135m%s\x1b[37m alts\n", SUCCESS, formatNumber(int64(len(alts))))
	fmt.Printf("%s Successfully Started Sniper At %s\n\n", INFO, time.Now().Add(time.Hour).Format("15:04:05"))

	var rateLimited, retryAfter = checkRateLimit()
	
	if rateLimited { fmt.Printf("%s Rate limited, retry in %s seconds\n", ERROR, retryAfter ); return }

	go func() { for _, altsArray := range splitAlts { 
		for _, alt := range altsArray {
			go claimer(alt); time.Sleep(time.Millisecond * 500)
		}
	} } ();

	go func() { for { saveInvites(); time.Sleep(time.Minute * 2) } }()
	go func() { for { runtime.GC(); time.Sleep(time.Second * 30) } }()
	go watchConfigChanges(); go watchTokenChanges()

	go func() { <-c; fmt.Printf("\r%s Tsukuyomi stopped, exiting after %s attempts...%s\n", ERROR, formatNumber(int64(atomic.LoadUint64(&totalAttempts))), strings.Repeat(" ", 95)); os.Exit(0) }()
	
	if !config.Sniper.NoInfo {
		go func() { for { for _, spinner := range []string { "/", "-", "\\", "|" } { runtime.ReadMemStats(&memory); fmt.Printf("[\x1b[37m\x1b[38;5;135m%s\x1b[37m] Sniping %s Servers | %s/%s Alts | %s Messages | %s Invites | %s Attempts | %s Sniped | %s Missed | %s Invalid | %s MB/S%s\r", spinner, formatNumber(int64(atomic.LoadUint64(&totalServers))), formatNumber(int64(amountReady)), formatNumber(int64(len(alts) - deadTokens)), formatNumber(int64(atomic.LoadUint64(&totalMessages))), formatNumber(int64(atomic.LoadUint64(&totalInvites))), formatNumber(int64(atomic.LoadUint64(&totalAttempts))), formatNumber(int64(totalSniped)), formatNumber(int64(totalMissed)), formatNumber(int64(totalInvalid)), formatNumber(int64(bToMb(memory.Alloc))), strings.Repeat(" ", 25)); time.Sleep(time.Millisecond * 150) } } }()
	}

	fmt.Scanln()
}