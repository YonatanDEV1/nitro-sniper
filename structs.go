package main

import (
	"encoding/json"
	"time"
)

type configStruct struct {
	Discord struct {
		APIVersion string `json:"apiVersion"`
		Webhooks struct {
			Successful string `json:"successful"`
			Missed     string `json:"missed"`
			EmbedMedia string `json:"embedMedia"`
		} `json:"webhooks"`
	} `json:"discord"`
	Sniper struct {
		SaveInvites             bool   `json:"saveInvites"`
		SnipeOnMain             bool   `json:"snipeOnMain"`
		NoInfo                  bool   `json:"noInfo"`
		Threads                 int    `json:"threads"`
	} `json:"sniper"`
}

type nitroStruct struct {
	ID               string      `json:"id"`
	SkuID            string      `json:"sku_id"`
	ApplicationID    string      `json:"application_id"`
	UserID           string      `json:"user_id"`
	PromotionID      interface{} `json:"promotion_id"`
	Type             int         `json:"type"`
	Deleted          bool        `json:"deleted"`
	GiftCodeFlags    int         `json:"gift_code_flags"`
	Consumed         bool        `json:"consumed"`
	GifterUserID     string      `json:"gifter_user_id"`
	SubscriptionPlan struct {
		ID            string      `json:"id"`
		Name          string      `json:"name"`
		Interval      int         `json:"interval"`
		IntervalCount int         `json:"interval_count"`
		TaxInclusive  bool        `json:"tax_inclusive"`
		SkuID         string      `json:"sku_id"`
		Currency      string      `json:"currency"`
		Price         int         `json:"price"`
		PriceTier     interface{} `json:"price_tier"`
	} `json:"subscription_plan"`
	Sku struct {
		ID             string        `json:"id"`
		Type           int           `json:"type"`
		DependentSkuID interface{}   `json:"dependent_sku_id"`
		ApplicationID  string        `json:"application_id"`
		ManifestLabels interface{}   `json:"manifest_labels"`
		AccessType     int           `json:"access_type"`
		Name           string        `json:"name"`
		Features       []interface{} `json:"features"`
		ReleaseDate    interface{}   `json:"release_date"`
		Premium        bool          `json:"premium"`
		Slug           string        `json:"slug"`
		Flags          int           `json:"flags"`
		ShowAgeGate    bool          `json:"show_age_gate"`
	} `json:"sku"`
	StoreListing        struct {
		Sku     struct {
			Name               string      `json:"name"`
		} `json:"sku"`
	} `json:"store_listing"`
}

type SysInfo struct {
    RAM      uint64 `bson:ram`
}

type Event struct {
	Operation int             `json:"op"`
	Sequence  int64           `json:"s"`
	Type      string          `json:"t"`
	RawData   json.RawMessage `json:"d"`
	Struct interface{} `json:"-"`
}

type Identify struct {
	Token        string `json:"token"`
	Capabilities int    `json:"capabilities"`
	Properties   struct {
		OS                     string      `json:"os"`
		Browser                string      `json:"browser"`
		Device                 string      `json:"device"`
		SystemLocale           string      `json:"system_locale"`
		BrowserUserAgent       string      `json:"browser_user_agent"`
		BrowserVersion         string      `json:"browser_version"`
		OsVersion              string      `json:"os_version"`
		Referrer               string      `json:"referrer"`
		ReferringDomain        string      `json:"referring_domain"`
		ReferrerCurrent        string      `json:"referrer_current"`
		ReferringDomainCurrent string      `json:"referring_domain_current"`
		ReleaseChannel         string      `json:"release_channel"`
		ClientBuildNumber      int         `json:"client_build_number"`
		ClientEventSource      interface{} `json:"client_event_source"`
		DesignID               int         `json:"design_id"`
	} `json:"properties"`
	Presence struct {
		Status     string        `json:"status"`
		Since      int           `json:"since"`
		Activities []interface{} `json:"activities"`
		Afk        bool          `json:"afk"`
	} `json:"presence"`
	Compress    bool `json:"compress"`
	ClientState struct {
		GuildVersions struct {
		} `json:"guild_versions"`
		HighestLastMessageID     string `json:"highest_last_message_id"`
		ReadStateVersion         int    `json:"read_state_version"`
		UserGuildSettingsVersion int    `json:"user_guild_settings_version"`
		UserSettingsVersion      int    `json:"user_settings_version"`
		PrivateChannelsVersion   string `json:"private_channels_version"`
		APICodeVersion           int    `json:"api_code_version"`
	} `json:"client_state"`
}

type helloOp struct {
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
}

type identifyStruct struct {
	Op   int      `json:"op"`
	Data Identify `json:"d"`
}

type heartbeatOp struct {
	Op   int   `json:"op"`
	Data int64 `json:"d"`
}

type Ready struct {
	User struct {
		Username      string `json:"username"`
		Discriminator string `json:"discriminator"`
	} `json:"user"`
	SessionID        string `json:"session_id"`
	ResumeGatewayURL string `json:"resume_gateway_url"`
	Guilds []struct {
		Properties struct {
			ID string `json:"id"`
		} `json:"properties"`
	} `json:"guilds"`
}

type messageCreateStruct struct {
	Content string `json:"content"`
	Author  struct {
		Username      string `json:"username"`
		Discriminator string `json:"discriminator"`
	} `json:"author"`
	GuildID string `json:"guild_id"`
}

type resumeStruct struct {
	Op   int `json:"op"`
	Data struct {
		Token     string `json:"token"`
		SessionID string `json:"session_id"`
		Sequence  int64  `json:"seq"`
	} `json:"d"`
}