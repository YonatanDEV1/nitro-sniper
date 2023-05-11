package main

import (
	"github.com/klauspost/compress/zlib"
	"github.com/gorilla/websocket"
	"github.com/json-iterator/go"
	_ "encoding/json"
	"sync/atomic"
	"math/rand"
	"net/http"
	"strings"
	"context"
	"errors"
	"bytes"
	"time"
	"fmt"
	"io"
)

func (session *Session) Connect(ctx context.Context) error {
	var err error

	session.Lock(); defer session.Unlock()

	if session.socketConnection != nil { return errors.New("Connection already exists") }
	var gatewayHeaders = http.Header{}; gatewayHeaders.Add("accept-encoding", "zlib")

	session.socketConnection, _, err = session.dialer.DialContext(ctx ,fmt.Sprintf("%s?encoding=json&v=%s", session.gateway, "10"), gatewayHeaders)

	if err != nil {
		session.Close(ctx)

		return err
	}

	session.socketConnection.SetCloseHandler(func(code int, text string) error {
		if code == 4004 {
			deadTokens += 1
		}

		return nil
	})

	defer func() {
		if err != nil {
			session.socketConnection.Close()
			session.socketConnection = nil
		}
	}()

	session.RateLimiter.Reset()

	session.listening = make(chan interface{})

	go session.Listen(ctx, session.socketConnection, session.listening)

	return err
}

func (session *Session) Listen(ctx context.Context, socketConnection *websocket.Conn, listening <-chan interface{}) {
	for {
		messageType, message, err := socketConnection.ReadMessage()

		if err != nil {
			session.RLock()
			sameConnection := session.socketConnection == socketConnection
			session.RUnlock()

			if sameConnection {
				session.Close(ctx)
				session.Reconnect(ctx)
			}

			return
		}

		select {
			case <-listening:
				return

			default:
				session.onEvent(ctx, messageType, message)
		}
	}
}

func (session *Session) onEvent(ctx context.Context, messageType int, message []byte) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var reader io.Reader; reader = bytes.NewBuffer(message)

	if messageType == websocket.BinaryMessage {
		z, error := zlib.NewReader(reader)
		if error != nil { return }

		defer func() { z.Close() }()

		reader = z
	}

	var e *Event; decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&e); err != nil { return }

	if e.Operation == 10 {
		var h helloOp; if err := json.Unmarshal(e.RawData, &h); err != nil { return }

		session.heartbeatInterval = time.Duration(h.HeartbeatInterval) * time.Millisecond
		session.heartbeatAck = time.Now().UTC()

		go session.Heartbeat()
	}

	if e.Operation == 1 {
		err := session.Send(ctx, heartbeatOp{1, atomic.LoadInt64(session.sequence)}); if err != nil {}
	}

	if e.Operation == 7 {
		session.closeConnection(context.TODO(), websocket.CloseServiceRestart)
		session.Reconnect(ctx)
	}

	if e.Operation == 9 {
		session.Identify()
	}

	if e.Operation == 10 {
		session.Identify()
	}

	if e.Operation == 11 {
		session.Lock(); session.heartbeatAck = time.Now().UTC(); session.Unlock()
	}

	if (e.Type == `READY`) {
		var r Ready
		err := json.Unmarshal(e.RawData, &r)

		if err == nil {
			session.gateway = r.ResumeGatewayURL

			if !session.isConnected {
				amountReady += 1

				atomic.AddUint64(&totalServers, uint64(len(r.Guilds)));

				session.account = fmt.Sprintf("%s#%s", r.User.Username, r.User.Discriminator)
				session.sessionId = r.SessionID
			}

			session.isConnected = true
		}
	}

	if (e.Type == `MESSAGE_CREATE`) {
		var messageCreateStruct messageCreateStruct

		err := json.Unmarshal(e.RawData, &messageCreateStruct);

		if err == nil {
			if successKey.Match([]byte(messageCreateStruct.Content)) {
				var startTime time.Time = time.Now(); var gift []string = strings.Split(messageCreateStruct.Content, "/")
				var giftId string = strings.Split(strings.Split(gift[len(gift)-1], "\n")[0], " ")[0];

				if len(giftId) >= 16 {
					if !alreadyClaimedNitro(giftId) {
						var snipeStatusCode, snipeResponseBody, snipeTime = snipeNitro(giftId, startTime); atomic.AddUint64(&totalAttempts, 1);
						var detectedAccount string = session.account; var guild string; if len(messageCreateStruct.GuildID) > 2 { guild = messageCreateStruct.GuildID } else { guild = "DMs" }

						switch {
							case snipeStatusCode == 0:
								fmt.Printf("%s [%ss] [%s] Error - %s%s\n", ERROR, convertMilliseconds(snipeTime), detectedAccount, snipeResponseBody, strings.Repeat(" ", 70))

							case snipeStatusCode == 429:
								if strings.Contains(snipeResponseBody, "global") {
									fmt.Printf("%s [%ss] [%s] Global Rate Limit - %s in %s%s\n", ERROR, convertMilliseconds(snipeTime), detectedAccount, giftId, guild, strings.Repeat(" ", 70));
								} else {
									fmt.Printf("%s [%ss] [%s] Token Rate Limit - %s in %s%s\n", ERROR, convertMilliseconds(snipeTime), detectedAccount, giftId, guild, strings.Repeat(" ", 70));
								}

								if len(config.Discord.Webhooks.Missed) > 20 {
									go discordPost(config.Discord.Webhooks.Missed, fmt.Sprintf(missedDiscordTemplate, missedDescriptions[rand.Intn(len(missedDescriptions))], time.Now().Format(time.RFC3339), fmt.Sprintf("`%s`", giftId), fmt.Sprintf("`%ss`", convertMilliseconds(snipeTime)), fmt.Sprintf("`%s`", detectedAccount), fmt.Sprintf("`%s#%s`", messageCreateStruct.Author.Username, messageCreateStruct.Author.Discriminator), fmt.Sprintf("`%s`", guild), fmt.Sprintf("`%s`", guild), fmt.Sprintf("```%s```", strings.ReplaceAll(snipeResponseBody, `"`, `\"`)), discordEmbedPicture, fmt.Sprintf("XO | %s | %s", vpsHostname, claimToken[len(claimToken)-5:]), discordEmbedPicture))
								}

							case snipeStatusCode == 404:
								fmt.Printf("%s [%ss] [%s] Invalid %s in %s%s\n", ERROR, convertMilliseconds(snipeTime), detectedAccount, giftId, guild, strings.Repeat(" ", 70)); totalInvalid += 1

							case snipeStatusCode == 400:
								if len(config.Discord.Webhooks.Missed) > 20 {
									go discordPost(config.Discord.Webhooks.Missed, fmt.Sprintf(missedDiscordTemplate, missedDescriptions[rand.Intn(len(missedDescriptions))], time.Now().Format(time.RFC3339), fmt.Sprintf("`%s`", giftId), fmt.Sprintf("`%ss`", convertMilliseconds(snipeTime)), fmt.Sprintf("`%s`", detectedAccount), fmt.Sprintf("`%s#%s`", messageCreateStruct.Author.Username, messageCreateStruct.Author.Discriminator), fmt.Sprintf("`%s`", guild), fmt.Sprintf("`%s`", guild), fmt.Sprintf("```%s```", strings.ReplaceAll(snipeResponseBody, `"`, `\"`)), discordEmbedPicture, fmt.Sprintf("XO | %s | %s", vpsHostname, claimToken[len(claimToken)-5:]), discordEmbedPicture))
								}

								fmt.Printf("%s [%ss] [%s] Missed %s in %s%s\n", ERROR, convertMilliseconds(snipeTime), detectedAccount, giftId, guild, strings.Repeat(" ", 70))
								totalMissed += 1

							case snipeStatusCode == 401:
								fmt.Printf("%s [%ss] [%s] Unauthorized %s in %s%s\n", ERROR, convertMilliseconds(snipeTime), detectedAccount, giftId, guild, strings.Repeat(" ", 70))

								if len(config.Discord.Webhooks.Missed) > 20 {
									go discordPost(config.Discord.Webhooks.Missed, fmt.Sprintf(missedDiscordTemplate, missedDescriptions[rand.Intn(len(missedDescriptions))], time.Now().Format(time.RFC3339), fmt.Sprintf("`%s`", giftId), fmt.Sprintf("`%ss`", convertMilliseconds(snipeTime)), fmt.Sprintf("`%s`", detectedAccount), fmt.Sprintf("`%s#%s`", messageCreateStruct.Author.Username, messageCreateStruct.Author.Discriminator), fmt.Sprintf("`%s`", guild), fmt.Sprintf("`%s`", guild), fmt.Sprintf("```%s```", strings.ReplaceAll(snipeResponseBody, `"`, `\"`)), discordEmbedPicture, fmt.Sprintf("XO | %s | %s", vpsHostname, claimToken[len(claimToken)-5:]), discordEmbedPicture))
								}  

							case snipeStatusCode == 403:
								fmt.Printf("%s [%ss] [%s] Account Locked %s in %s%s\n", ERROR, convertMilliseconds(snipeTime), detectedAccount, giftId, guild, strings.Repeat(" ", 70))

								if len(config.Discord.Webhooks.Missed) > 20 {
									go discordPost(config.Discord.Webhooks.Missed, fmt.Sprintf(missedDiscordTemplate, missedDescriptions[rand.Intn(len(missedDescriptions))], time.Now().Format(time.RFC3339), fmt.Sprintf("`%s`", giftId), fmt.Sprintf("`%ss`", convertMilliseconds(snipeTime)), fmt.Sprintf("`%s`", detectedAccount), fmt.Sprintf("`%s#%s`", messageCreateStruct.Author.Username, messageCreateStruct.Author.Discriminator), fmt.Sprintf("`%s`", guild), fmt.Sprintf("`%s`", guild), fmt.Sprintf("```%s```", strings.ReplaceAll(snipeResponseBody, `"`, `\"`)), discordEmbedPicture, fmt.Sprintf("XO | %s | %s", vpsHostname, claimToken[len(claimToken)-5:]), discordEmbedPicture))
								}

							case snipeStatusCode == 200:
								totalSniped += 1;

								var claimResponse nitroStruct
				    			err := json.Unmarshal([]byte(snipeResponseBody), &claimResponse)

				    			if err == nil {
				    				if len(claimResponse.SubscriptionPlan.Name) >= 3 {
					    				fmt.Printf("%s [%ss] [%s] Successfully sniped %s - %s in %s%s\n", SUCCESS, convertMilliseconds(snipeTime), detectedAccount, claimResponse.SubscriptionPlan.Name, giftId, guild, strings.Repeat(" ", 70))

										if len(config.Discord.Webhooks.Successful) > 20 {
											go discordPost(config.Discord.Webhooks.Successful, fmt.Sprintf(successfulDiscordTemplate, claimedDescriptions[rand.Intn(len(claimedDescriptions))], time.Now().Format(time.RFC3339), fmt.Sprintf("`%s`", giftId), fmt.Sprintf("`%ss`", convertMilliseconds(snipeTime)), fmt.Sprintf("`%s`", claimResponse.SubscriptionPlan.Name), fmt.Sprintf("`%s`", claimResponse.GifterUserID), fmt.Sprintf("`%s`", guild), fmt.Sprintf("`%s`", guild), discordEmbedPicture, fmt.Sprintf("XO | %s | %s", vpsHostname, claimToken[len(claimToken)-5:]), discordEmbedPicture))
										}
					    			} else if len(claimResponse.StoreListing.Sku.Name) >= 3 {
					    				fmt.Printf("%s [%ss] [%s] Successfully sniped %s - %s in %s%s\n", SUCCESS, convertMilliseconds(snipeTime), detectedAccount, claimResponse.StoreListing.Sku.Name, giftId, guild, strings.Repeat(" ", 70))

										if len(config.Discord.Webhooks.Successful) > 20 {
											go discordPost(config.Discord.Webhooks.Successful, fmt.Sprintf(successfulDiscordTemplate, claimedDescriptions[rand.Intn(len(claimedDescriptions))], time.Now().Format(time.RFC3339), fmt.Sprintf("`%s`", giftId), fmt.Sprintf("`%ss`", convertMilliseconds(snipeTime)), fmt.Sprintf("`%s`", claimResponse.StoreListing.Sku.Name), fmt.Sprintf("`%s#%s`", messageCreateStruct.Author.Username, messageCreateStruct.Author.Discriminator), fmt.Sprintf("`%s`", guild), fmt.Sprintf("`%s`", guild), discordEmbedPicture, fmt.Sprintf("XO | %s | %s", vpsHostname, claimToken[len(claimToken)-5:]), discordEmbedPicture))
										}
					    			} else {
					    				fmt.Printf("%s [%ss] [%s] Successfully sniped %s - %s in %s%s\n", SUCCESS, convertMilliseconds(snipeTime), detectedAccount, "Unknown", giftId, guild, strings.Repeat(" ", 70))

										if len(config.Discord.Webhooks.Successful) > 20 {
											go discordPost(config.Discord.Webhooks.Successful, fmt.Sprintf(successfulDiscordTemplate, claimedDescriptions[rand.Intn(len(claimedDescriptions))], time.Now().Format(time.RFC3339), fmt.Sprintf("`%s`", giftId), fmt.Sprintf("`%ss`", convertMilliseconds(snipeTime)), fmt.Sprintf("`%s`", "Unknown"), fmt.Sprintf("`%s#%s`", messageCreateStruct.Author.Username, messageCreateStruct.Author.Discriminator), fmt.Sprintf("`%s`", guild), fmt.Sprintf("`%s`", guild), discordEmbedPicture, fmt.Sprintf("XO | %s | %s", vpsHostname, claimToken[len(claimToken)-5:]), discordEmbedPicture))
										}
					    			}
				    			}
						}
					}
				}
			}

			if inviteSuccessKey.Match([]byte(messageCreateStruct.Content)) {
				var splitInvites = strings.Split(messageCreateStruct.Content, "\n")

				for _, invite := range splitInvites {
					if inviteSuccessKey.Match([]byte(invite)) {
						invites = append(invites, strings.ReplaceAll(inviteSuccessKey.FindStringSubmatch(invite)[1], "https", ""))
						atomic.AddUint64(&totalInvites, 1);
					}
				}
			}
		}

		atomic.AddUint64(&totalMessages, 1);
	}

	atomic.StoreInt64(session.sequence, e.Sequence)
}

func (session *Session) Identify() {
	if session.sessionId == "" {
		var Identify Identify

		Identify.Token = session.token
		Identify.Compress = true
		Identify.Capabilities = 8189

		Identify.Properties.OS = "Mac OS X"
		Identify.Properties.Browser = "Chrome"
		Identify.Properties.SystemLocale = "en-GB"
		Identify.Properties.BrowserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36"
		Identify.Properties.BrowserVersion = "112.0.0.0"
		Identify.Properties.OsVersion = "10.15.7"
		Identify.Properties.ReleaseChannel = "stable"
		Identify.Properties.ClientBuildNumber = 196590
		Identify.Properties.DesignID = 0

		Identify.Presence.Status = "online"
		Identify.Presence.Since = 0
		Identify.Presence.Afk = false

		Identify.ClientState.HighestLastMessageID = "0"
		Identify.ClientState.ReadStateVersion = 0
		Identify.ClientState.UserGuildSettingsVersion = -1
		Identify.ClientState.UserSettingsVersion = -1
		Identify.ClientState.PrivateChannelsVersion = "0"
		Identify.ClientState.APICodeVersion = 0

		identityData := identifyStruct{2, Identify}

		err := session.Send(context.TODO(), identityData)
		if err != nil {}
	} else {
		p := resumeStruct{}

		p.Op = 6
		p.Data.Token = session.token
		p.Data.SessionID = session.sessionId
		p.Data.Sequence = atomic.LoadInt64(session.sequence)

		err := session.Send(context.TODO(), p)
		if err != nil {}
	}
}

func (session *Session) Heartbeat() {
	heartbeatTicker := time.NewTicker(session.heartbeatInterval); defer heartbeatTicker.Stop()

	if session.listening == nil || session.socketConnection == nil { return }

	for {
		select {
			case <-session.listening:
				return

			case <-heartbeatTicker.C:
				session.sendHeartbeat()
		}
	}
}

func (session *Session) sendHeartbeat() {
	ctx, cancel := context.WithTimeout(context.Background(), session.heartbeatInterval)
	defer cancel()

	if session.listening == nil || session.socketConnection == nil { return }

	session.RLock(); last := session.heartbeatAck; session.RUnlock()

	session.socketMutex.Lock()
	err := session.Send(context.Background(), heartbeatOp{1, atomic.LoadInt64(session.sequence)}); if err != nil {}
	session.heartbeatSent = time.Now().UTC()
	session.socketMutex.Unlock()

	if err != nil || time.Now().UTC().Sub(last) > (session.heartbeatInterval * failedHeartbeatAcks) {
		session.Close(ctx)
		session.Reconnect(ctx)
	}
}

func (session *Session) Reconnect(ctx context.Context) {
	timer := time.NewTimer(time.Duration(time.Second * 30)); defer timer.Stop()

	select {
		case <-ctx.Done():
			timer.Stop()

		case <-timer.C:
			err := session.Connect(ctx)
			if err != nil {}
	}
}

func (session *Session) Close(ctx context.Context) {
	session.closeConnection(ctx, websocket.CloseNormalClosure)
}

func (session *Session) closeConnection(ctx context.Context, closeCode int) {
	session.Lock()

	if session.listening != nil {
		close(session.listening)
		session.listening = nil
	}

	if session.socketConnection != nil {
		session.socketMutex.Lock()

		session.RateLimiter.Close(ctx)

		err := session.socketConnection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, "")); if err != nil {}
		session.socketMutex.Unlock()

		time.Sleep(1 * time.Second)

		session.Close(ctx);
		session.socketConnection = nil

		if closeCode == websocket.CloseNormalClosure {
			session.gateway = "wss://gateway.discord.gg"
			session.sequence = new(int64)
			session.sessionId = ""
		}
	}

	session.Unlock()
}

func (session *Session) Send(ctx context.Context, i interface{}) error {
	session.socketMutex.Lock()
	defer session.socketMutex.Unlock()

	if err := session.RateLimiter.Wait(ctx); err != nil {
		return errors.New("Nope")
	}

	defer session.RateLimiter.Unlock()
	return session.socketConnection.WriteJSON(i);
}