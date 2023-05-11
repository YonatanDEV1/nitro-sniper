package main

import (
	"github.com/valyala/fasthttp"
	"net/http"
	"runtime"
	"regexp"
	"time"
)

// Miscellaneous
var (
	SUCCESS string = "[\x1b[37m\x1b[38;5;135m+\x1b[37m]"
	INFO string = "[\x1b[37m\x1b[38;5;135m?\x1b[37m]"
	ERROR string = "[\x1b[31m-\x1b[39m]"

	memory runtime.MemStats

	sniperAscii string =
"\n" + "\x1b[37m\x1b[38;5;135m     .--'''''''''--.\n" +
"\x1b[37m\x1b[38;5;135m   .'      .---.      '.\n" +
"\x1b[37m\x1b[38;5;135m  /    .-----------.    \\\n" +
"\x1b[37m\x1b[38;5;135m /        .-----.        \\\n" +
"\x1b[37m\x1b[38;5;135m |       .-.   .-.       |\n" +
"\x1b[37m\x1b[38;5;135m |      /   \\ /   \\      |" + "\x1b[37m" + "    Tsukuyomi\n" +
"\x1b[37m\x1b[38;5;135m  \\    | .-. | .-. |    / " + "\x1b[37m" + "     %s\n" +
"\x1b[37m\x1b[38;5;135m   '-._| | | | | | |_.-' " + "\x1b[37m" + "           \n" +
"\x1b[37m\x1b[38;5;135m       | '-' | '-' |\n" +
"\x1b[37m\x1b[38;5;135m        \\___/ \\___/     \x1b[37mThank you - XO\n" +
"\x1b[37m\x1b[38;5;135m     _.-'  /   \\  `-._\n" +
"\x1b[37m\x1b[38;5;135m   .' _.--|     |--._ '. " + "\x1b[37m" + "   <-- Niggas when dey lose their nitro's\n" +
"\x1b[37m\x1b[38;5;135m   ' _...-|     |-..._ '\n" +
"\x1b[37m\x1b[38;5;135m          |     |\n"
)

// HTTP
var (
	webhookClient *fasthttp.Client // Important shit here : )
    discordClient *http.Client // Important shit here : )
    apiClient *fasthttp.Client // Important shit here : )
    
    claimRequestHeaders http.Header // Have header object ready as it'll be faster than building one per request
)

// Sniper Utility
var (
	sniperVersion string = "The end"

	vpsHostname string
	discordId string
	
	config configStruct

	totalAttempts uint64
	totalMessages uint64
	totalInvites uint64
	totalServers uint64
	
	totalInvalid int
	totalSniped int
	amountReady int
	totalMissed int
	deadTokens int

	splitAlts [][]string
	invites []string
	claimed []string
	missed []string
	alts []string
)

// Discord
var (
	successKey = regexp.MustCompile("(?i)(cord.com/gifts/|cordapp.com/gifts/|cord.gift/)([a-zA-Z0-9]+)") // Don't check for whole "discord" as the regex will index each char so it's faster like this
	inviteSuccessKey = regexp.MustCompile("cord.gg/([0-9a-zA-Z]+)|cord.com/invites/([0-9a-zA-Z]+)") // Don't check for whole "discord" as the regex will index each char so it's faster like this
	
	failedHeartbeatAcks time.Duration = 5 * time.Millisecond

	missedDiscordTemplate string = `{"embeds":[{"title":"Tsukuyomi","description":":x: %s","timestamp":"%s","color":"9662683","fields":[{"name":"Code","value":"%s","inline":true},{"name":"Delay","value":"%s","inline":true},{"name":"Sniper","value":"%s","inline":true},{"name":"Sender","value":"%s","inline":true},{"name":"Guild ID","value":"%s","inline":true},{"name":"Guild Name","value":"%s","inline":true},{"name":"Response","value":"%s","inline":false}],"thumbnail":{"url":"%s"},"footer":{"text":"%s","icon_url":"%s"}}]}`
	successfulDiscordTemplate string = `{"embeds":[{"title":"Tsukuyomi","description":":white_check_mark: %s","timestamp":"%s","color":"9662683","fields":[{"name":"Code","value":"%s","inline":true},{"name":"Delay","value":"%s","inline":true},{"name":"Type","value":"%s","inline":true},{"name":"Sender","value":"%s","inline":true},{"name":"Guild ID","value":"%s","inline":true}, {"name":"Guild Name","value":"%s","inline":true}],"thumbnail":{"url":"%s"},"footer":{"text":"%s","icon_url":"%s"}}]}`

	discordEmbedPicture string
	apiVersion string
	claimToken string

	discordHost string = "discord"
	userAgent string = "Discord/196590"

	gatewayVersion string = "10"

	claimedDescriptions = []string {
		"Is it a bird? Is it a plane? No, it's just Tsukuyomi detecting another nitro! :cowboy:",
		"What happened to Velocity? :thinking:",
		"Tsukuyomi detected a nitro :money_mouth:",
		"This just in, a nitro has been dropped! :newspaper2:",
		"Well, well, well. What do we have here then? :mag:",
		"It would be a shame if they lost that... :pensive:",
		"And... Anotha one :point_up::skin-tone-1:",
		"Ah shit, here we go again :rolling_eyes:",
		"I was not expecting that... :flushed:",
		"Anotha one :point_up::skin-tone-1:",
		"This dumbass really sent it :joy:",
		"I didn't see that coming :eyes:",
		"This just in, a nitro has been sent! :newspaper2:",
		"It seems Tsukuyomi was yet again 2fast4u!",
		"I feel sorry for whoever pressed send :face_with_peeking_eye:",
		"Someone warn these people I'm getting tired of this :yawning_face:",
		"Someone sent a gift but it seems they didn't want it:dizzy:",
	}

	missedDescriptions = []string {
		"Maybe they will send another, who knows? :fingers_crossed::skin-tone-1:",
		"Why am I so slow :pensive:",
		"I'm sorry, I'll do better next time :heart_hands:",
		"Don't give up on me, I won't let you down :pensive:",
		":index_pointing_at_the_viewer: I'm just warming up",
		":face_with_raised_eyebrow:If this happens again feel free to just turn me off",
		"I'm starting to question my abilities:broken_heart:",
	}
)